package service

import (
	"regexp"
	"strings"

	"blackbox-api/internal/models"
)

func EnrichDeviceMetadata(device *models.Device, siteMappings map[string]string) {
	device.DeviceType = ExtractDeviceType(device.Hostname)
	device.SiteCode = ExtractSiteCode(device.Hostname)
	if device.SiteCode != "" {
		if location, ok := siteMappings[device.SiteCode]; ok {
			device.Location = location
		}
	}
}

func ExtractDeviceType(hostname string) string {
	trimmed := strings.TrimSpace(hostname)
	if trimmed == "" {
		return ""
	}
	re := regexp.MustCompile(`^[A-Za-z]+`)
	return strings.ToUpper(re.FindString(trimmed))
}

func ExtractSiteCode(hostname string) string {
	trimmed := strings.ToLower(strings.TrimSpace(hostname))
	if trimmed == "" {
		return ""
	}
	re := regexp.MustCompile(`\.([a-z0-9]+)-config$`)
	match := re.FindStringSubmatch(trimmed)
	if len(match) != 2 {
		return ""
	}
	return match[1]
}
