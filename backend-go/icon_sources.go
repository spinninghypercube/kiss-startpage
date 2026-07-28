package main

import "strings"

type iconSourceDefinition struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Provider    string `json:"provider"`
	Prefix      string `json:"prefix,omitempty"`
	LicenseName string `json:"licenseName"`
	LicenseURL  string `json:"licenseUrl,omitempty"`
}

type iconProviderDefinition struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
	LicenseName string `json:"licenseName,omitempty"`
	LicenseURL  string `json:"licenseUrl,omitempty"`
}

var iconProviderRegistry = []iconProviderDefinition{
	{
		ID:          "iconify",
		Label:       "Iconify",
		Description: "Fourteen curated open-source icon collections.",
		LicenseName: "Varies by collection",
		LicenseURL:  "https://icon-sets.iconify.design/",
	},
	{
		ID:          "selfhst",
		Label:       "selfh.st/icons",
		Description: "Technology and self-hosted application icons.",
		LicenseName: "CC BY 4.0",
		LicenseURL:  "https://github.com/selfhst/icons/blob/main/LICENSE",
	},
	{
		ID:          "dashboard",
		Label:       "Dashboard Icons",
		Description: "Curated service and application icons for dashboards.",
		LicenseName: "Apache-2.0",
		LicenseURL:  "https://github.com/homarr-labs/dashboard-icons/blob/main/LICENSE",
	},
	{
		ID:          "local",
		Label:       "Local icons",
		Description: "Private icon files stored on this KISS server.",
	},
	{
		ID:          "website",
		Label:       "Website icon",
		Description: "Favicons discovered from the button URLs.",
	},
	{
		ID:          "wikimedia",
		Label:       "Wikimedia Commons",
		Description: "Strictly filtered CC0 and public-domain SVG artwork.",
		LicenseName: "CC0 / Public domain only",
		LicenseURL:  "https://commons.wikimedia.org/wiki/Commons:Licensing",
	},
}

var iconSourceRegistry = []iconSourceDefinition{
	{
		ID:          "selfhst",
		Label:       "selfh.st/icons",
		Provider:    "selfhst",
		LicenseName: "Varies by icon",
		LicenseURL:  "https://selfh.st/icons/",
	},
	{
		ID:          "iconify-simple",
		Label:       "Iconify · Simple Icons",
		Provider:    "iconify",
		Prefix:      "simple-icons",
		LicenseName: "CC0-1.0",
		LicenseURL:  "https://github.com/simple-icons/simple-icons/blob/develop/LICENSE.md",
	},
	{
		ID:          "iconify-logos",
		Label:       "Iconify · SVG Logos",
		Provider:    "iconify",
		Prefix:      "logos",
		LicenseName: "CC0-1.0",
		LicenseURL:  "https://raw.githubusercontent.com/gilbarbara/logos/master/LICENSE.txt",
	},
	{
		ID:          "iconify-mdi",
		Label:       "Iconify · Material Design Icons",
		Provider:    "iconify",
		Prefix:      "mdi",
		LicenseName: "Apache-2.0",
		LicenseURL:  "https://github.com/Templarian/MaterialDesign/blob/master/LICENSE",
	},
	{
		ID:          "iconify-material-symbols",
		Label:       "Iconify · Material Symbols",
		Provider:    "iconify",
		Prefix:      "material-symbols",
		LicenseName: "Apache-2.0",
		LicenseURL:  "https://github.com/google/material-design-icons/blob/master/LICENSE",
	},
	{
		ID:          "iconify-tabler",
		Label:       "Iconify · Tabler Icons",
		Provider:    "iconify",
		Prefix:      "tabler",
		LicenseName: "MIT",
		LicenseURL:  "https://github.com/tabler/tabler-icons/blob/master/LICENSE",
	},
	{
		ID:          "iconify-lucide",
		Label:       "Iconify · Lucide",
		Provider:    "iconify",
		Prefix:      "lucide",
		LicenseName: "ISC",
		LicenseURL:  "https://github.com/lucide-icons/lucide/blob/main/LICENSE",
	},
	{
		ID:          "iconify-phosphor",
		Label:       "Iconify · Phosphor",
		Provider:    "iconify",
		Prefix:      "ph",
		LicenseName: "MIT",
		LicenseURL:  "https://github.com/phosphor-icons/core/blob/main/LICENSE",
	},
	{
		ID:          "iconify-fa6-solid",
		Label:       "Iconify · Font Awesome 6 Solid",
		Provider:    "iconify",
		Prefix:      "fa6-solid",
		LicenseName: "CC-BY-4.0",
		LicenseURL:  "https://creativecommons.org/licenses/by/4.0/",
	},
	{
		ID:          "iconify-fluent",
		Label:       "Iconify · Fluent UI System Icons",
		Provider:    "iconify",
		Prefix:      "fluent",
		LicenseName: "MIT",
		LicenseURL:  "https://github.com/microsoft/fluentui-system-icons/blob/main/LICENSE",
	},
	{
		ID:          "iconify-carbon",
		Label:       "Iconify · Carbon",
		Provider:    "iconify",
		Prefix:      "carbon",
		LicenseName: "Apache-2.0",
	},
	{
		ID:          "iconify-bootstrap",
		Label:       "Iconify · Bootstrap Icons",
		Provider:    "iconify",
		Prefix:      "bi",
		LicenseName: "MIT",
		LicenseURL:  "https://github.com/twbs/icons/blob/main/LICENSE",
	},
	{
		ID:          "iconify-devicon",
		Label:       "Iconify · Devicon",
		Provider:    "iconify",
		Prefix:      "devicon",
		LicenseName: "MIT",
		LicenseURL:  "https://github.com/devicons/devicon/blob/master/LICENSE",
	},
	{
		ID:          "iconify-heroicons",
		Label:       "Iconify · Heroicons",
		Provider:    "iconify",
		Prefix:      "heroicons",
		LicenseName: "MIT",
		LicenseURL:  "https://github.com/tailwindlabs/heroicons/blob/master/LICENSE",
	},
	{
		ID:          "iconify-remix",
		Label:       "Iconify · Remix Icon",
		Provider:    "iconify",
		Prefix:      "ri",
		LicenseName: "Apache-2.0",
		LicenseURL:  "https://github.com/Remix-Design/RemixIcon/blob/master/License",
	},
}

func iconSources() []iconSourceDefinition {
	items := make([]iconSourceDefinition, len(iconSourceRegistry))
	copy(items, iconSourceRegistry)
	return items
}

func iconProviders() []iconProviderDefinition {
	items := make([]iconProviderDefinition, len(iconProviderRegistry))
	copy(items, iconProviderRegistry)
	return items
}

func iconProviderByID(id string) (iconProviderDefinition, bool) {
	normalized := strings.ToLower(strings.TrimSpace(id))
	for _, provider := range iconProviderRegistry {
		if provider.ID == normalized {
			return provider, true
		}
	}
	return iconProviderDefinition{}, false
}

func defaultIconProviderIDs() []string {
	items := make([]string, 0, len(iconProviderRegistry))
	for _, provider := range iconProviderRegistry {
		items = append(items, provider.ID)
	}
	return items
}

func iconifyPrefixes() []string {
	items := []string{}
	for _, source := range iconSourceRegistry {
		if source.Provider == "iconify" {
			items = append(items, source.Prefix)
		}
	}
	return items
}

func iconSourceByID(id string) (iconSourceDefinition, bool) {
	normalized := strings.ToLower(strings.TrimSpace(id))
	for _, source := range iconSourceRegistry {
		if source.ID == normalized {
			return source, true
		}
	}
	return iconSourceDefinition{}, false
}

func iconSourceByPrefix(prefix string) (iconSourceDefinition, bool) {
	normalized := strings.ToLower(strings.TrimSpace(prefix))
	for _, source := range iconSourceRegistry {
		if source.Provider == "iconify" && source.Prefix == normalized {
			return source, true
		}
	}
	return iconSourceDefinition{}, false
}
