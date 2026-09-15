package instance

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// ParseIPPorts 解析 IP/端口表达式为 "IP:端口" 列表
//
// 支持的输入（逗号分隔）:
//   - 单个 IP:            10.1.255.38
//   - 短范围:             10.1.255.24-26
//   - 完整范围:           10.1.26.5-10.1.26.7
//   - CIDR:               10.1.255.0/24
//   - 任意项带端口:       10.1.255.38:443、10.1.255.24-26:80、10.1.255.0/24:8080
//
// 返回:
//   []string - 展开后的 "IP:端口"（无端口时为纯 IP）列表
//   error    - 解析错误
func ParseIPPorts(input string) ([]string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("IP 表达式不能为空")
	}

	var result []string
	for _, network := range strings.Split(input, ",") {
		network = strings.TrimSpace(network)
		if network == "" {
			continue
		}

		// 先尝试按最后一个冒号拆分 IP 与端口（兼容 IPv4，无冒号则端口为空）
		ipPart := network
		port := ""
		if idx := strings.LastIndex(network, ":"); idx >= 0 {
			candidatePort := network[idx+1:]
			// 仅当冒号后为纯数字时才视为端口，避免误切 IPv6 或非法输入
			if isAllDigits(candidatePort) {
				ipPart = network[:idx]
				port = candidatePort
			}
		}

		ips, err := expandIPPart(ipPart)
		if err != nil {
			return nil, err
		}

		for _, ip := range ips {
			if port != "" {
				result = append(result, ip+":"+port)
			} else {
				result = append(result, ip)
			}
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("未能解析出任何 IP")
	}
	return result, nil
}

// expandIPPart 展开单个 IP 段（范围 / CIDR / 单 IP）
func expandIPPart(ipPart string) ([]string, error) {
	ipPart = strings.TrimSpace(ipPart)
	if ipPart == "" {
		return nil, fmt.Errorf("IP 段为空")
	}

	// 范围：a-b（支持短写 24-26 或完整 10.1.26.5-10.1.26.7）
	if strings.Contains(ipPart, "-") {
		return expandRange(ipPart)
	}

	// CIDR
	if strings.Contains(ipPart, "/") {
		ip, ipnet, err := net.ParseCIDR(ipPart)
		if err != nil {
			return nil, fmt.Errorf("非法 CIDR %q: %w", ipPart, err)
		}
		_ = ip
		var ips []string
		for cur := ipnet.IP.Mask(ipnet.Mask); ipnet.Contains(cur); cur = nextIP(cur) {
			ips = append(ips, cur.String())
		}
		// 去掉网络地址本身（与 Python net.hosts() 行为一致，仅当前缀 < /31）
		ones, bits := ipnet.Mask.Size()
		if bits-ones > 1 && len(ips) > 2 {
			ips = ips[1 : len(ips)-1]
		}
		return ips, nil
	}

	// 单个 IP
	if net.ParseIP(ipPart) == nil {
		return nil, fmt.Errorf("非法 IP %q", ipPart)
	}
	return []string{ipPart}, nil
}

// expandRange 展开 IP 范围
func expandRange(ipPart string) ([]string, error) {
	parts := strings.SplitN(ipPart, "-", 2)
	startStr := strings.TrimSpace(parts[0])
	endStr := strings.TrimSpace(parts[1])

	startIP := net.ParseIP(startStr)
	if startIP == nil {
		return nil, fmt.Errorf("非法起始 IP %q", startStr)
	}

	var endIP net.IP
	if net.ParseIP(endStr) != nil {
		// 完整结束 IP
		endIP = net.ParseIP(endStr)
	} else {
		// 短写：仅最后一段，如 10.1.255.24-26
		last, err := strconv.Atoi(endStr)
		if err != nil {
			return nil, fmt.Errorf("非法结束 IP %q", endStr)
		}
		base := startIP.To4()
		if base == nil {
			return nil, fmt.Errorf("仅支持 IPv4 范围: %q", ipPart)
		}
		lastOctet := last
		endIP = net.IPv4(base[0], base[1], base[2], byte(lastOctet))
	}

	start4 := startIP.To4()
	end4 := endIP.To4()
	if start4 == nil || end4 == nil {
		return nil, fmt.Errorf("仅支持 IPv4 范围: %q", ipPart)
	}
	if compareIP(start4, end4) > 0 {
		return nil, fmt.Errorf("起始 IP 大于结束 IP: %q", ipPart)
	}

	var ips []string
	for cur := start4; compareIP(cur, end4) <= 0; cur = nextIP(cur) {
		ips = append(ips, cur.String())
	}
	return ips, nil
}

// nextIP 返回下一个 IP（原地累加尾字节，处理进位）
func nextIP(ip net.IP) net.IP {
	next := make(net.IP, len(ip))
	copy(next, ip)
	for i := len(next) - 1; i >= 0; i-- {
		next[i]++
		if next[i] != 0 {
			break
		}
	}
	return next
}

// compareIP 比较两个等长 IP（-1/0/1）
func compareIP(a, b net.IP) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}

// isAllDigits 判断字符串是否全为数字
func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
