(function () {
  "use strict";

  const STORAGE_KEY = "homelabDashboardConfigV1";
  const LAST_TAB_KEY = "homelabDashboardLastTab";

  const defaultConfig = {
    tabs: [
      { id: "external", label: "External" },
      { id: "internal", label: "Internal" }
    ],
    groups: [
      {
        id: "group-smarthome",
        title: "01. Smart-Home",
        groupEnd: true,
        entries: [
          { id: "entry-searxng", name: "SearXNG", icon: "searxng.svg", iconData: "", links: { external: "https://search.example.local/", internal: "http://192.168.68.98" } },
          { id: "entry-opnsense", name: "OPNsense", icon: "opnsense.svg", iconData: "", links: { external: "https://opnsense.example.local/", internal: "http://192.168.68.1" } },
          { id: "entry-proxmox-router", name: "Proxmox-router", icon: "proxmox.svg", iconData: "", links: { external: "https://proxmox-router.example.local/", internal: "https://192.168.68.10:8006/" } },
          { id: "entry-proxmox-homelab", name: "Proxmox-homelab", icon: "proxmox.svg", iconData: "", links: { external: "https://proxmox.example.local/", internal: "https://192.168.68.11:8006/" } },
          { id: "entry-pbs-router", name: "PBS Router", icon: "pbs.svg", iconData: "", links: { external: "https://pbs.example.local/", internal: "https://192.168.68.20:8007/" } },
          { id: "entry-pbs-homelab", name: "PBS Homelab", icon: "pbs.svg", iconData: "", links: { external: "https://pbs-homelab.example.local/", internal: "https://192.168.68.21:8007/" } },
          { id: "entry-cyberpanel", name: "CyberPanel", icon: "hosting.svg", iconData: "", links: { external: "https://cyberpanel.example.local/", internal: "https://192.168.68.65:8090" } },
          { id: "entry-synology", name: "Synology", icon: "dsm.svg", iconData: "", links: { external: "https://dsm.example.local/", internal: "https://192.168.68.40:5001" } },
          { id: "entry-joplin", name: "Joplin", icon: "joplin.svg", iconData: "", links: { external: "https://joplin.example.local/", internal: "https://joplin.example.local/" } }
        ]
      },
      {
        id: "group-core-infra",
        title: "02. Core-infrastructure",
        groupEnd: true,
        entries: [
          { id: "entry-homeassistant", name: "Home Assistant", icon: "home-assistant.svg", iconData: "", links: { external: "https://ha.example.local/", internal: "http://192.168.68.60:8123/" } },
          { id: "entry-pihole", name: "Pihole", icon: "pihole.svg", iconData: "", links: { external: "https://pihole.example.local/", internal: "http://192.168.68.30/admin/" } },
          { id: "entry-adguard", name: "Adguard", icon: "adguard.svg", iconData: "", links: { external: "https://adguard.example.local/", internal: "http://192.168.68.31/" } },
          { id: "entry-authentik", name: "Authentik", icon: "authentik.svg", iconData: "", links: { external: "https://auth.example.local/", internal: "http://192.168.68.93:9000" } },
          { id: "entry-nginx", name: "Nginx", icon: "nginx.svg", iconData: "", links: { external: "https://nginx.example.local/", internal: "http://192.168.68.86:81" } },
          { id: "entry-wgeasy", name: "WG-Easy", icon: "wireguard.svg", iconData: "", links: { external: "https://wgeasy.example.local/", internal: "http://192.168.68.87:51821" } },
          { id: "entry-vaultwarden", name: "Vaultwarden", icon: "vaultwarden.svg", iconData: "", links: { external: "https://vault.example.local/", internal: "https://192.168.68.92:8000" } }
        ]
      },
      {
        id: "group-monitoring",
        title: "04. Monitoring",
        groupEnd: true,
        entries: [
          { id: "entry-grafana", name: "Grafana", icon: "grafana.svg", iconData: "", links: { external: "https://grafana.example.local/", internal: "http://192.168.68.76:3000/" } },
          { id: "entry-myspeed", name: "Myspeed", icon: "myspeed.svg", iconData: "", links: { external: "https://myspeed.example.local/", internal: "http://192.168.68.90:5216/" } },
          { id: "entry-prometheus", name: "Prometheus", icon: "prometheus.svg", iconData: "", links: { external: "https://prometheus.example.local/targets", internal: "http://192.168.68.74:9090/targets" } },
          { id: "entry-uptime-kuma", name: "Uptime Kuma", icon: "uptime-kuma.svg", iconData: "", links: { external: "https://uptimekuma.example.local/", internal: "http://192.168.68.81:3001/" } },
          { id: "entry-lodgy", name: "Lodgy", icon: "lodgy.png", iconData: "", links: { external: "https://lodgy.example.local/", internal: "http://192.168.68.61:8080/" } }
        ]
      },
      {
        id: "group-download-upload",
        title: "05. Download-upload",
        groupEnd: true,
        entries: [
          { id: "entry-qbittorrent", name: "qBittorrent", icon: "qbittorrent.svg", iconData: "", links: { external: "https://qbittorrent.example.local/", internal: "http://192.168.68.57:8090/#/" } },
          { id: "entry-filebrowser", name: "FileBrowser", icon: "filebrowser.svg", iconData: "", links: { external: "https://filebrowser.example.local/", internal: "http://192.168.68.63:8080" } },
          { id: "entry-filebrowserq", name: "FileBrowserQ", icon: "filebrowser-quantum.svg", iconData: "", links: { external: "https://filebrowserq.example.local/", internal: "http://192.168.68.70:8080" } }
        ]
      },
      {
        id: "group-arr-stack",
        title: "06. Arr-stack",
        groupEnd: true,
        entries: [
          { id: "entry-bazarr", name: "Bazarr", icon: "bazarr.svg", iconData: "", links: { external: "https://bazarr.example.local/", internal: "http://192.168.68.73:6767" } },
          { id: "entry-prowlarr", name: "Prowlarr", icon: "prowlarr.svg", iconData: "", links: { external: "https://prowlarr.example.local/", internal: "http://192.168.68.69:9696" } },
          { id: "entry-radarr", name: "Radarr", icon: "radarr.svg", iconData: "", links: { external: "https://radarr.example.local/", internal: "http://192.168.68.71:7878" } },
          { id: "entry-readarr", name: "Readarr", icon: "readarr.svg", iconData: "", links: { external: "https://readarr.example.local/", internal: "http://192.168.68.79:8787" } },
          { id: "entry-sonarr", name: "Sonarr", icon: "sonarr.svg", iconData: "", links: { external: "https://sonarr.example.local/", internal: "http://192.168.68.72:8989" } },
          { id: "entry-overseerr", name: "Overseerr", icon: "overseerr.svg", iconData: "", links: { external: "https://overseerr.example.local/", internal: "http://192.168.68.94:5055" } }
        ]
      },
      {
        id: "group-media-servers",
        title: "07. Media-servers",
        groupEnd: true,
        entries: [
          { id: "entry-plex-intel-nuc", name: "Plex Intel NUC", icon: "plex.svg", iconData: "", links: { external: "https://plex.example.local/web/index.html#!/", internal: "http://192.168.68.64:32400/web/index.html#!" } },
          { id: "entry-plex-router", name: "Plex-router", icon: "plex.svg", iconData: "", links: { external: "https://plex-router.example.local/web/index.html#!/", internal: "http://192.168.68.55:32400/web/index.html#!" } },
          { id: "entry-jellyfin", name: "Jellyfin", icon: "jellyfin.svg", iconData: "", links: { external: "https://jellyfin.example.local/", internal: "http://192.168.68.68:8096/web/#/home.html" } },
          { id: "entry-calibre", name: "Calibre", icon: "calibre.svg", iconData: "", links: { external: "https://calibre.example.local/", internal: "http://192.168.68.84:8080" } },
          { id: "entry-calibre-web", name: "Calibre-Web", icon: "calibre-web.svg", iconData: "", links: { external: "https://calibre-web.example.local/", internal: "http://192.168.68.83:8083" } },
          { id: "entry-tautulli", name: "Tautulli", icon: "tautulli.svg", iconData: "", links: { external: "https://tautulli.example.local/", internal: "http://192.168.68.95:8181" } },
          { id: "entry-tautulli-router", name: "Tautulli-router", icon: "tautulli.svg", iconData: "", links: { external: "https://tautulli-router.example.local/", internal: "http://192.168.68.96:8181" } }
        ]
      }
    ]
  };

  function clone(value) {
    return JSON.parse(JSON.stringify(value));
  }

  function createId(prefix) {
    return `${prefix}-${Math.random().toString(36).slice(2, 10)}`;
  }

  function makeSafeTabId(rawValue) {
    const source = (rawValue || "tab").toString().trim().toLowerCase();
    const safe = source
      .replace(/[^a-z0-9]+/g, "-")
      .replace(/^-+|-+$/g, "")
      .slice(0, 32);
    return safe || "tab";
  }

  function makeUniqueTabId(baseId, usedIds) {
    if (!usedIds.has(baseId)) {
      usedIds.add(baseId);
      return baseId;
    }

    let counter = 2;
    let candidate = `${baseId}-${counter}`;
    while (usedIds.has(candidate)) {
      counter += 1;
      candidate = `${baseId}-${counter}`;
    }
    usedIds.add(candidate);
    return candidate;
  }

  function normalizeTabs(inputTabs) {
    const sourceTabs = Array.isArray(inputTabs) && inputTabs.length ? inputTabs : clone(defaultConfig.tabs);
    const usedIds = new Set();

    const tabs = sourceTabs
      .map((tab) => {
        const fallbackLabel = (tab && typeof tab.label === "string" && tab.label.trim()) || (tab && tab.id) || "Tab";
        const requestedId = makeSafeTabId(tab && (tab.id || tab.label));
        const uniqueId = makeUniqueTabId(requestedId, usedIds);
        return {
          id: uniqueId,
          label: fallbackLabel.toString().trim()
        };
      })
      .filter((tab) => tab.label.length > 0);

    if (!tabs.length) {
      return clone(defaultConfig.tabs);
    }

    return tabs;
  }

  function normalizeEntry(entry, tabs) {
    const links = {};
    tabs.forEach((tab) => {
      const existingLink = entry && entry.links && typeof entry.links[tab.id] === "string" ? entry.links[tab.id] : "";
      links[tab.id] = existingLink;
    });

    if (entry && typeof entry.external === "string" && links.external === undefined) {
      links.external = entry.external;
    }
    if (entry && typeof entry.internal === "string" && links.internal === undefined) {
      links.internal = entry.internal;
    }

    return {
      id: entry && entry.id ? String(entry.id) : createId("entry"),
      name: entry && typeof entry.name === "string" ? entry.name : "New Entry",
      icon: entry && typeof entry.icon === "string" ? entry.icon : "",
      iconData: entry && typeof entry.iconData === "string" ? entry.iconData : "",
      links
    };
  }

  function normalizeGroup(group, tabs) {
    const entries = Array.isArray(group && group.entries) ? group.entries.map((entry) => normalizeEntry(entry, tabs)) : [];

    return {
      id: group && group.id ? String(group.id) : createId("group"),
      title: group && typeof group.title === "string" ? group.title : "New Group",
      groupEnd: Boolean(group && group.groupEnd),
      entries
    };
  }

  function normalizeConfig(inputConfig) {
    const config = inputConfig && typeof inputConfig === "object" ? inputConfig : clone(defaultConfig);
    const tabs = normalizeTabs(config.tabs);

    const sourceGroups = Array.isArray(config.groups) && config.groups.length ? config.groups : clone(defaultConfig.groups);
    const groups = sourceGroups.map((group) => normalizeGroup(group, tabs));

    return {
      tabs,
      groups
    };
  }

  function loadConfig() {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      if (!raw) {
        return normalizeConfig(defaultConfig);
      }
      const parsed = JSON.parse(raw);
      return normalizeConfig(parsed);
    } catch (error) {
      console.error("Failed to load dashboard config, using default.", error);
      return normalizeConfig(defaultConfig);
    }
  }

  function saveConfig(config) {
    const normalized = normalizeConfig(config);
    localStorage.setItem(STORAGE_KEY, JSON.stringify(normalized));
    return normalized;
  }

  function resetConfig() {
    localStorage.removeItem(STORAGE_KEY);
    return normalizeConfig(defaultConfig);
  }

  function getLastTab() {
    return localStorage.getItem(LAST_TAB_KEY) || "";
  }

  function setLastTab(tabId) {
    localStorage.setItem(LAST_TAB_KEY, tabId);
  }

  window.DashboardData = {
    STORAGE_KEY,
    LAST_TAB_KEY,
    defaultConfig: clone(defaultConfig),
    loadConfig,
    saveConfig,
    resetConfig,
    normalizeConfig,
    makeSafeTabId,
    createId,
    getLastTab,
    setLastTab
  };
})();
