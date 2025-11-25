package dispatcher

import (
	"fmt"
	commnet "koria-core/common/net"
	v2config "koria-core/config/v2"
	"log"
	"net"
	"regexp"
	"strconv"
	"strings"
)

// Router управляет маршрутизацией на основе правил
type Router struct {
	rules []RoutingRule
}

// RoutingRule внутреннее представление правила маршрутизации
type RoutingRule struct {
	domainPatterns []*regexp.Regexp
	ipCIDRs        []*net.IPNet
	portRanges     []PortRange
	network        string // "tcp", "udp", ""
	outboundTag    string
}

// PortRange диапазон портов
type PortRange struct {
	start uint16
	end   uint16
}

// NewRouter создает новый роутер из конфигурации
func NewRouter(config *v2config.RoutingConfig) (*Router, error) {
	if config == nil {
		return &Router{rules: []RoutingRule{}}, nil
	}

	router := &Router{
		rules: make([]RoutingRule, 0, len(config.Rules)),
	}

	for _, ruleConfig := range config.Rules {
		rule, err := parseRoutingRule(ruleConfig)
		if err != nil {
			log.Printf("[Router] Warning: failed to parse rule: %v", err)
			continue
		}
		router.rules = append(router.rules, rule)
	}

	log.Printf("[Router] Loaded %d routing rules", len(router.rules))
	return router, nil
}

// parseRoutingRule парсит правило из конфига
func parseRoutingRule(config v2config.RoutingRule) (RoutingRule, error) {
	rule := RoutingRule{
		outboundTag: config.OutboundTag,
		network:     config.Network,
	}

	// Парсим domain patterns
	for _, pattern := range config.Domain {
		regex, err := domainPatternToRegex(pattern)
		if err != nil {
			return rule, fmt.Errorf("invalid domain pattern %s: %w", pattern, err)
		}
		rule.domainPatterns = append(rule.domainPatterns, regex)
	}

	// Парсим IP CIDRs
	for _, cidr := range config.IP {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			// Попробуем как одиночный IP
			ip := net.ParseIP(cidr)
			if ip == nil {
				return rule, fmt.Errorf("invalid IP/CIDR %s: %w", cidr, err)
			}
			// Создаем /32 или /128 CIDR
			if ip.To4() != nil {
				_, ipNet, _ = net.ParseCIDR(cidr + "/32")
			} else {
				_, ipNet, _ = net.ParseCIDR(cidr + "/128")
			}
		}
		rule.ipCIDRs = append(rule.ipCIDRs, ipNet)
	}

	// Парсим port ranges
	if config.Port != "" {
		ranges, err := parsePortRanges(config.Port)
		if err != nil {
			return rule, fmt.Errorf("invalid port specification %s: %w", config.Port, err)
		}
		rule.portRanges = ranges
	}

	return rule, nil
}

// domainPatternToRegex конвертирует domain pattern в regex
func domainPatternToRegex(pattern string) (*regexp.Regexp, error) {
	// Поддерживаемые паттерны:
	// "example.com" - точное совпадение
	// "*.example.com" - wildcard субдомены
	// "domain:example.com" - домен и все субдомены
	// "regexp:^.*\.example\.com$" - полный regex
	// "full:example.com" - только точное совпадение

	if strings.HasPrefix(pattern, "regexp:") {
		regexStr := strings.TrimPrefix(pattern, "regexp:")
		return regexp.Compile(regexStr)
	}

	if strings.HasPrefix(pattern, "full:") {
		domain := strings.TrimPrefix(pattern, "full:")
		return regexp.Compile("^" + regexp.QuoteMeta(domain) + "$")
	}

	if strings.HasPrefix(pattern, "domain:") {
		domain := strings.TrimPrefix(pattern, "domain:")
		// Совпадает domain и все субдомены
		escapedDomain := regexp.QuoteMeta(domain)
		return regexp.Compile("(^|\\.)" + escapedDomain + "$")
	}

	// Обработка wildcard
	if strings.HasPrefix(pattern, "*.") {
		domain := strings.TrimPrefix(pattern, "*.")
		escapedDomain := regexp.QuoteMeta(domain)
		// Совпадает любой субдомен, но не сам домен
		return regexp.Compile("^[^.]+\\." + escapedDomain + "$")
	}

	// Точное совпадение или wildcard в середине
	if strings.Contains(pattern, "*") {
		// Заменяем * на .*
		regexStr := regexp.QuoteMeta(pattern)
		regexStr = strings.ReplaceAll(regexStr, "\\*", ".*")
		return regexp.Compile("^" + regexStr + "$")
	}

	// Простое точное совпадение
	return regexp.Compile("^" + regexp.QuoteMeta(pattern) + "$")
}

// parsePortRanges парсит спецификацию портов
// Примеры: "80", "80,443", "8080-8090", "80,443,8000-9000"
func parsePortRanges(portSpec string) ([]PortRange, error) {
	var ranges []PortRange

	parts := strings.Split(portSpec, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)

		if strings.Contains(part, "-") {
			// Диапазон портов
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid port range: %s", part)
			}

			start, err := strconv.ParseUint(strings.TrimSpace(rangeParts[0]), 10, 16)
			if err != nil {
				return nil, fmt.Errorf("invalid start port: %s", rangeParts[0])
			}

			end, err := strconv.ParseUint(strings.TrimSpace(rangeParts[1]), 10, 16)
			if err != nil {
				return nil, fmt.Errorf("invalid end port: %s", rangeParts[1])
			}

			if start > end {
				return nil, fmt.Errorf("invalid port range: start > end")
			}

			ranges = append(ranges, PortRange{start: uint16(start), end: uint16(end)})
		} else {
			// Одиночный порт
			port, err := strconv.ParseUint(part, 10, 16)
			if err != nil {
				return nil, fmt.Errorf("invalid port: %s", part)
			}
			ranges = append(ranges, PortRange{start: uint16(port), end: uint16(port)})
		}
	}

	return ranges, nil
}

// MatchOutbound возвращает тег outbound для destination
func (r *Router) MatchOutbound(dest commnet.Destination) string {
	for _, rule := range r.rules {
		if r.matchRule(rule, dest) {
			log.Printf("[Router] Matched rule -> %s for %s", rule.outboundTag, dest.String())
			return rule.outboundTag
		}
	}

	log.Printf("[Router] No rule matched for %s, using default", dest.String())
	return "" // Пустой тег = default outbound
}

// matchRule проверяет совпадает ли destination с правилом
func (r *Router) matchRule(rule RoutingRule, dest commnet.Destination) bool {
	// Проверка network (tcp/udp)
	if rule.network != "" && string(dest.Network) != rule.network {
		return false
	}

	// Проверка port
	if len(rule.portRanges) > 0 {
		matched := false
		for _, portRange := range rule.portRanges {
			if dest.Port >= portRange.start && dest.Port <= portRange.end {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Если есть domain patterns - проверяем domain
	if len(rule.domainPatterns) > 0 {
		matched := false
		for _, pattern := range rule.domainPatterns {
			if pattern.MatchString(dest.Address) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Если есть IP CIDRs - проверяем IP
	if len(rule.ipCIDRs) > 0 {
		// Резолвим адрес в IP (если это не IP)
		ip := net.ParseIP(dest.Address)
		if ip == nil {
			// Это hostname, не IP - не совпадает с IP правилом
			return false
		}

		matched := false
		for _, cidr := range rule.ipCIDRs {
			if cidr.Contains(ip) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Если нет никаких условий - правило всегда совпадает (default)
	if len(rule.domainPatterns) == 0 && len(rule.ipCIDRs) == 0 && len(rule.portRanges) == 0 && rule.network == "" {
		return true
	}

	// Все условия совпали
	return true
}
