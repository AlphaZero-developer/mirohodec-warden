(function () {
  const themeKey = "laya-panel-theme";
  const panelSettingsKey = "laya-panel-settings";
  const root = document.documentElement;
  const colorVars = [
    "--bg",
    "--surface",
    "--surface-soft",
    "--surface-strong",
    "--text",
    "--muted",
    "--line",
    "--line-soft",
    "--accent",
    "--accent-strong",
    "--accent-soft",
    "--info",
    "--info-soft",
    "--danger",
    "--danger-soft",
    "--warning",
    "--warning-soft"
  ];
  const colorPresetValues = [
    "#111416",
    "#191f22",
    "#20282c",
    "#2a3439",
    "#eef3f2",
    "#a6b2b0",
    "#344145",
    "#37b7a4",
    "#6bd6c5",
    "#173833",
    "#86a8e7",
    "#1c2a45",
    "#ff6b86",
    "#421c27",
    "#e7ae4f",
    "#3d2f18"
  ];
  const defaultPanelSettings = {
    transition: "fade",
    motionSpeed: "normal",
    hotReload: "on",
    backgroundPreset: "clean",
    backgroundImage: "",
    backgroundBlur: "0px",
    surfaceOpacity: "100%",
    surfaceBlur: "0px",
    rightPanel: "hover",
    fontFamily: "",
    fontUrl: "",
    fontScale: "normal",
    colors: {}
  };

  function text(value) {
    return (value || "").replace(/[&<>"]/g, function (char) {
      return {"&": "&amp;", "<": "&lt;", ">": "&gt;", "\"": "&quot;"}[char];
    });
  }

  function preferredTheme() {
    const saved = localStorage.getItem(themeKey);
    if (saved === "light" || saved === "dark") return saved;
    return window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
  }

  function applyTheme(theme) {
    root.dataset.theme = theme;
    localStorage.setItem(themeKey, theme);
    document.querySelectorAll("[data-theme-toggle]").forEach(function (button) {
      button.textContent = theme === "dark" ? "Светлая" : "Тёмная";
      button.setAttribute("aria-label", theme === "dark" ? "Включить светлую тему" : "Включить тёмную тему");
    });
  }

  applyTheme(preferredTheme());

  function cloneDefaultPanelSettings() {
    return {
      transition: defaultPanelSettings.transition,
      motionSpeed: defaultPanelSettings.motionSpeed,
      hotReload: defaultPanelSettings.hotReload,
      backgroundPreset: defaultPanelSettings.backgroundPreset,
      backgroundImage: defaultPanelSettings.backgroundImage,
      backgroundBlur: defaultPanelSettings.backgroundBlur,
      surfaceOpacity: defaultPanelSettings.surfaceOpacity,
      surfaceBlur: defaultPanelSettings.surfaceBlur,
      rightPanel: defaultPanelSettings.rightPanel,
      fontFamily: defaultPanelSettings.fontFamily,
      fontUrl: defaultPanelSettings.fontUrl,
      fontScale: defaultPanelSettings.fontScale,
      colors: {}
    };
  }

  function readPanelSettings() {
    let parsed = {};
    try {
      parsed = JSON.parse(localStorage.getItem(panelSettingsKey) || "{}") || {};
    } catch (err) {
      parsed = {};
    }
    return {
      ...cloneDefaultPanelSettings(),
      ...parsed,
      colors: {...(parsed.colors || {})}
    };
  }

  let panelSettings = readPanelSettings();
  let pendingPanelSettings = {...panelSettings, colors: {...panelSettings.colors}};

  function validHex(value) {
    return /^#[0-9a-f]{6}$/i.test(value || "");
  }

  function colorToHex(value) {
    value = (value || "").trim();
    if (/^#[0-9a-f]{6}$/i.test(value)) return value.toLowerCase();
    if (/^#[0-9a-f]{3}$/i.test(value)) {
      return "#" + value.slice(1).split("").map(function (part) { return part + part; }).join("").toLowerCase();
    }
    const rgb = value.match(/^rgba?\((\d+),\s*(\d+),\s*(\d+)/i);
    if (!rgb) return "";
    return "#" + [rgb[1], rgb[2], rgb[3]].map(function (part) {
      return Math.max(0, Math.min(255, Number(part))).toString(16).padStart(2, "0");
    }).join("");
  }

  function computedPanelColor(name) {
    return colorToHex(getComputedStyle(root).getPropertyValue(name)) || "#000000";
  }

  function normalizePanelSettings(settings) {
    const next = {...cloneDefaultPanelSettings(), ...(settings || {})};
    next.transition = ["fade", "slide-left", "slide-right", "slide-up", "slide-down", "zoom", "scale-soft", "flip", "blur", "rotate", "diagonal", "none"].includes(next.transition) ? next.transition : "fade";
    next.motionSpeed = ["fast", "normal", "slow"].includes(next.motionSpeed) ? next.motionSpeed : "normal";
    next.hotReload = next.hotReload === "off" ? "off" : "on";
    next.backgroundPreset = ["clean", "aurora", "nebula", "frost", "image"].includes(next.backgroundPreset) ? next.backgroundPreset : "clean";
    next.backgroundImage = safePanelURL(next.backgroundImage);
    next.backgroundBlur = ["0px", "4px", "10px", "18px", "28px"].includes(next.backgroundBlur) ? next.backgroundBlur : "0px";
    next.surfaceOpacity = ["100%", "94%", "88%", "78%", "66%"].includes(next.surfaceOpacity) ? next.surfaceOpacity : "100%";
    next.surfaceBlur = ["0px", "8px", "16px", "24px"].includes(next.surfaceBlur) ? next.surfaceBlur : "0px";
    next.rightPanel = ["hover", "pinned", "off"].includes(next.rightPanel) ? next.rightPanel : "hover";
    next.fontScale = ["compact", "normal", "large"].includes(next.fontScale) ? next.fontScale : "normal";
    next.fontFamily = String(next.fontFamily || "").trim();
    next.fontUrl = String(next.fontUrl || "").trim();
    if (next.fontUrl && !/^https?:\/\//i.test(next.fontUrl)) next.fontUrl = "";
    next.colors = {...(next.colors || {})};
    Object.keys(next.colors).forEach(function (name) {
      if (!colorVars.includes(name) || !validHex(next.colors[name])) {
        delete next.colors[name];
      }
    });
    return next;
  }

  function safePanelURL(value) {
    value = String(value || "").trim();
    if (!value) return "";
    try {
      const url = new URL(value, window.location.href);
      if (url.protocol !== "http:" && url.protocol !== "https:") return "";
      return url.href;
    } catch (err) {
      return "";
    }
  }

  function cssURL(value) {
    return "url(\"" + String(value).replace(/["\\\n\r]/g, "") + "\")";
  }

  function writePanelSettings(settings) {
    localStorage.setItem(panelSettingsKey, JSON.stringify(normalizePanelSettings(settings)));
  }

  function fillPanelSettingsControls(settings) {
    document.querySelectorAll("[data-panel-setting]").forEach(function (input) {
      const name = input.dataset.panelSetting;
      const value = settings[name] ?? defaultPanelSettings[name] ?? "";
      if (input.value !== String(value)) input.value = String(value);
    });
    document.querySelectorAll("[data-panel-color]").forEach(function (input) {
      const name = input.dataset.panelColor;
      const value = settings.colors[name] || computedPanelColor(name);
      if (validHex(value) && input.value.toLowerCase() !== value.toLowerCase()) {
        input.value = value;
      }
    });
    syncColorPickerPreviews();
  }

  function enhanceColorPickers() {
    document.querySelectorAll("[data-panel-color]").forEach(function (input) {
      if (input.dataset.colorEnhanced === "true") return;
      input.dataset.colorEnhanced = "true";
      input.classList.add("color-hex-input");
      input.autocomplete = "off";
      input.inputMode = "text";
      input.maxLength = 7;
      input.placeholder = "#37b7a4";
      input.spellcheck = false;

      const card = input.closest("label");
      if (card) card.classList.add("color-card");

      const shell = document.createElement("div");
      shell.className = "color-picker-shell";
      const preview = document.createElement("button");
      preview.type = "button";
      preview.className = "color-preview-button";
      preview.dataset.colorPreview = input.dataset.panelColor;
      preview.title = "Выбрать цвет";
      input.parentNode.insertBefore(shell, input);
      shell.appendChild(preview);
      shell.appendChild(input);

      const presets = document.createElement("div");
      presets.className = "color-presets";
      colorPresetValues.forEach(function (value) {
        const button = document.createElement("button");
        button.type = "button";
        button.className = "color-preset";
        button.dataset.colorPreset = value;
        button.style.setProperty("--preset", value);
        button.title = value;
        presets.appendChild(button);
      });
      shell.after(presets);
    });
    syncColorPickerPreviews();
  }

  function syncColorPickerPreviews() {
    document.querySelectorAll("[data-panel-color]").forEach(function (input) {
      const value = validHex(input.value) ? input.value.toLowerCase() : computedPanelColor(input.dataset.panelColor);
      const preview = input.closest(".color-picker-shell")?.querySelector("[data-color-preview]");
      if (preview) {
        preview.style.setProperty("--picked", value);
        preview.setAttribute("aria-label", "Цвет " + value);
      }
      input.classList.toggle("invalid", input.value.trim() !== "" && !validHex(input.value));
    });
  }

  function applyPanelSettings(settings, fillControls) {
    settings = normalizePanelSettings(settings);
    root.dataset.transition = settings.transition;
    root.dataset.motionSpeed = settings.motionSpeed;
    root.dataset.fontScale = settings.fontScale;
    root.dataset.bgPreset = settings.backgroundPreset;
    root.dataset.rightPanel = settings.rightPanel;
    root.style.setProperty("--background-blur", settings.backgroundBlur);
    root.style.setProperty("--surface-alpha", settings.surfaceOpacity);
    root.style.setProperty("--panel-blur", settings.surfaceBlur);
    if (settings.backgroundImage) {
      root.style.setProperty("--panel-bg-image", cssURL(settings.backgroundImage));
    } else {
      root.style.removeProperty("--panel-bg-image");
    }
    if (settings.fontFamily) {
      root.style.setProperty("--panel-font", settings.fontFamily);
    } else {
      root.style.removeProperty("--panel-font");
    }
    let fontLink = document.getElementById("panel-font-link");
    if (settings.fontUrl) {
      if (!fontLink) {
        fontLink = document.createElement("link");
        fontLink.id = "panel-font-link";
        fontLink.rel = "stylesheet";
        document.head.appendChild(fontLink);
      }
      if (fontLink.href !== settings.fontUrl) fontLink.href = settings.fontUrl;
    } else if (fontLink) {
      fontLink.remove();
    }
    colorVars.forEach(function (name) {
      if (validHex(settings.colors[name])) {
        root.style.setProperty(name, settings.colors[name]);
      } else {
        root.style.removeProperty(name);
      }
    });
    if (fillControls) fillPanelSettingsControls(settings);
    syncColorPickerPreviews();
  }

  function collectPanelSettingsFromControls() {
    const next = {...pendingPanelSettings, colors: {...pendingPanelSettings.colors}};
    document.querySelectorAll("[data-panel-setting]").forEach(function (input) {
      next[input.dataset.panelSetting] = input.value;
    });
    document.querySelectorAll("[data-panel-color]").forEach(function (input) {
      if (validHex(input.value)) next.colors[input.dataset.panelColor] = input.value.toLowerCase();
    });
    return normalizePanelSettings(next);
  }

  function saveAndApplyPanelSettings(settings, fillControls) {
    panelSettings = normalizePanelSettings(settings);
    pendingPanelSettings = {...panelSettings, colors: {...panelSettings.colors}};
    writePanelSettings(panelSettings);
    applyPanelSettings(panelSettings, fillControls);
  }

  function handlePanelSettingsInput(event) {
    if (!event.target.closest("[data-panel-setting], [data-panel-color]")) return false;
    pendingPanelSettings = collectPanelSettingsFromControls();
    if (event.target.matches("[data-panel-setting='hotReload']") || pendingPanelSettings.hotReload === "on") {
      saveAndApplyPanelSettings(pendingPanelSettings, false);
    }
    syncColorPickerPreviews();
    return true;
  }

  function resetPanelSettings() {
    localStorage.removeItem(panelSettingsKey);
    panelSettings = cloneDefaultPanelSettings();
    pendingPanelSettings = cloneDefaultPanelSettings();
    applyPanelSettings(panelSettings, true);
    showToast("Настройки сброшены");
  }

  applyPanelSettings(panelSettings, false);

  function updatePreview(form) {
    const kind = form.dataset.editor;
    const preview = form.querySelector(".preview");
    if (!preview) return;

    if (kind === "quote") {
      const quote = form.querySelector("[name=text]")?.value || "";
      const author = form.querySelector("[name=author]")?.value || "";
      preview.innerHTML = text(quote) + (author ? "<br><small>" + text(author) + "</small>" : "");
      updateDuplicate(form, quote);
    }

    if (kind === "faq") {
      const answer = form.querySelector("[name=answer]")?.value || "";
      const patterns = form.querySelector("[name=patterns]")?.value || "";
      preview.innerHTML = text(answer);
      updateDuplicate(form, patterns);
    }
  }

  function updateDuplicate(form, value) {
    const note = form.querySelector(".dupe");
    if (!note) return;
    const currentID = form.querySelector("[name=id]")?.value || "";
    const normalized = (value || "").trim().toLowerCase();
    if (!normalized) {
      note.textContent = "";
      note.className = "dupe muted";
      return;
    }
    const hit = [...document.querySelectorAll("[data-value]")].some(function (node) {
      return (node.dataset.id || "") !== currentID && (node.dataset.value || "").trim().toLowerCase() === normalized;
    });
    note.textContent = hit ? "Похоже, такая запись уже есть." : "";
    note.className = hit ? "dupe error" : "dupe muted";
  }

  function ensureFilterTools(input) {
    if (input.dataset.enhanced === "true") return;
    input.dataset.enhanced = "true";
    const container = input.parentElement;
    if (!container) return;

    const status = document.createElement("span");
    status.className = "filter-status";
    status.dataset.filterStatus = "";
    container.appendChild(status);

    const clear = document.createElement("button");
    clear.type = "button";
    clear.className = "secondary";
    clear.dataset.filterClear = "";
    clear.textContent = "Очистить";
    container.appendChild(clear);
  }

  function updateFilter(input) {
    ensureFilterTools(input);
    const query = input.value.trim().toLowerCase();
    const nodes = [...document.querySelectorAll(input.dataset.filter)];
    let shown = 0;
    nodes.forEach(function (node) {
      const visible = node.textContent.toLowerCase().includes(query);
      node.style.display = visible ? "" : "none";
      if (visible) shown += 1;
    });
    const status = input.parentElement?.querySelector("[data-filter-status]");
    if (status) {
      status.textContent = query ? "Найдено: " + shown + " из " + nodes.length : "Всего: " + nodes.length;
    }
  }

  function updateAILogFilters(scope) {
    scope = scope || document.querySelector("[data-ai-log-filter]");
    if (!scope) return false;
    const query = (scope.querySelector("[data-ai-log-search]")?.value || "").trim().toLowerCase();
    const level = scope.querySelector("[data-ai-log-level]")?.value || "";
    const cards = [...document.querySelectorAll("[data-ai-log-card]")];
    let shown = 0;
    cards.forEach(function (card) {
      const levelOK = !level || card.dataset.aiLogLevel === level;
      const queryOK = !query || card.innerText.toLowerCase().includes(query);
      const visible = levelOK && queryOK;
      card.hidden = !visible;
      if (visible) shown += 1;
    });
    const count = scope.querySelector("[data-ai-log-count]");
    if (count) count.textContent = "Показано: " + shown + " из " + cards.length;
    return true;
  }

  function handleAILogFilterInput(event) {
    if (!event.target.closest("[data-ai-log-search], [data-ai-log-level]")) return false;
    updateAILogFilters(event.target.closest("[data-ai-log-filter]"));
    return true;
  }

  function setAILogDetails(open) {
    document.querySelectorAll("[data-ai-log-card]").forEach(function (card) {
      const details = card.querySelector("details");
      if (details && !card.hidden) details.open = open;
    });
  }

  function sortTable(table, columnIndex, th) {
    const tbody = table.tBodies[0];
    if (!tbody) return;
    const current = th.dataset.sortDir === "asc" ? "desc" : "asc";
    table.querySelectorAll("th[data-sort-dir]").forEach(function (node) {
      delete node.dataset.sortDir;
    });
    th.dataset.sortDir = current;

    const rows = [...tbody.rows].filter(function (row) {
      return row.cells.length > columnIndex && !row.querySelector(".empty");
    });
    rows.sort(function (a, b) {
      const av = cellSortValue(a.cells[columnIndex]);
      const bv = cellSortValue(b.cells[columnIndex]);
      const an = Number(av.replace(",", "."));
      const bn = Number(bv.replace(",", "."));
      const result = Number.isFinite(an) && Number.isFinite(bn)
        ? an - bn
        : av.localeCompare(bv, "ru", {numeric: true, sensitivity: "base"});
      return current === "asc" ? result : -result;
    });
    rows.forEach(function (row) { tbody.appendChild(row); });
  }

  function cellSortValue(cell) {
    return (cell?.innerText || "").trim().replace(/\s+/g, " ");
  }

  function enhanceTables() {
    document.querySelectorAll("table").forEach(function (table) {
      if (table.dataset.enhanced === "true") return;
      table.dataset.enhanced = "true";
      table.querySelectorAll("thead th").forEach(function (th, index) {
        th.dataset.sort = String(index);
        th.tabIndex = 0;
      });

      const panel = table.closest(".panel");
      const head = panel?.querySelector(".table-head");
      if (head && !head.querySelector("[data-export-csv]")) {
        const tools = document.createElement("div");
        tools.className = "table-tools";
        const exportButton = document.createElement("button");
        exportButton.type = "button";
        exportButton.className = "secondary";
        exportButton.dataset.exportCsv = "";
        exportButton.textContent = "CSV";
        tools.appendChild(exportButton);
        head.appendChild(tools);
      }
    });
  }

  function exportTable(button) {
    const table = button.closest(".panel")?.querySelector("table");
    if (!table) return;
    const rows = [...table.querySelectorAll("tr")].filter(function (row) {
      return row.offsetParent !== null && !row.querySelector(".empty");
    });
    const csv = rows.map(function (row) {
      return [...row.cells].map(function (cell) {
        return "\"" + cell.innerText.trim().replace(/\s+/g, " ").replace(/"/g, "\"\"") + "\"";
      }).join(",");
    }).join("\n");
    const blob = new Blob([csv], {type: "text/csv;charset=utf-8"});
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = "laya-panel.csv";
    link.click();
    URL.revokeObjectURL(url);
  }

  function showToast(message) {
    let toast = document.querySelector(".copy-toast");
    if (!toast) {
      toast = document.createElement("div");
      toast.className = "copy-toast";
      document.body.appendChild(toast);
    }
    toast.textContent = message;
    toast.classList.add("show");
    window.clearTimeout(showToast.timer);
    showToast.timer = window.setTimeout(function () {
      toast.classList.remove("show");
    }, 1400);
  }

  function copyValue(value) {
    value = (value || "").trim();
    if (!value) return;
    if (navigator.clipboard?.writeText) {
      navigator.clipboard.writeText(value).then(function () {
        showToast("Скопировано");
      }).catch(function () {
        fallbackCopy(value);
      });
      return;
    }
    fallbackCopy(value);
  }

  function fallbackCopy(value) {
    const input = document.createElement("textarea");
    input.value = value;
    input.style.position = "fixed";
    input.style.opacity = "0";
    document.body.appendChild(input);
    input.select();
    document.execCommand("copy");
    input.remove();
    showToast("Скопировано");
  }

  function maybeNavigateWithTransition(event) {
    const link = event.target.closest("a[href]");
    if (!link || event.defaultPrevented || event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return;
    if (link.target || link.hasAttribute("download")) return;
    const url = new URL(link.href, window.location.href);
    if (url.origin !== window.location.origin || url.pathname === window.location.pathname && url.search === window.location.search && url.hash) return;
    event.preventDefault();
    document.body.classList.add("is-leaving");
    const delay = root.dataset.transition === "none"
      ? 0
      : root.dataset.motionSpeed === "slow"
        ? 190
        : root.dataset.motionSpeed === "fast"
          ? 80
          : 110;
    window.setTimeout(function () {
      window.location.href = url.href;
    }, delay);
  }

  document.addEventListener("input", function (event) {
    if (handlePanelSettingsInput(event)) return;
    if (handleAILogFilterInput(event)) return;

    const filter = event.target.closest("[data-filter]");
    if (filter) updateFilter(filter);

    const editor = event.target.closest("[data-editor]");
    if (editor) updatePreview(editor);
  });

  document.addEventListener("change", function (event) {
    if (handleAILogFilterInput(event)) return;
  });

  document.addEventListener("click", function (event) {
    const colorPreset = event.target.closest("[data-color-preset]");
    if (colorPreset) {
      const input = colorPreset.closest(".color-card")?.querySelector("[data-panel-color]");
      if (input) {
        input.value = colorPreset.dataset.colorPreset;
        input.dispatchEvent(new Event("input", {bubbles: true}));
      }
      return;
    }

    const colorPreview = event.target.closest("[data-color-preview]");
    if (colorPreview) {
      const input = colorPreview.closest(".color-picker-shell")?.querySelector("[data-panel-color]");
      if (input) {
        input.focus();
        input.select();
      }
      return;
    }

    const themeToggle = event.target.closest("[data-theme-toggle]");
    if (themeToggle) {
      applyTheme(root.dataset.theme === "dark" ? "light" : "dark");
      applyPanelSettings(panelSettings, true);
      return;
    }

    const panelSettingsApply = event.target.closest("[data-panel-settings-apply]");
    if (panelSettingsApply) {
      saveAndApplyPanelSettings(collectPanelSettingsFromControls(), true);
      showToast("Настройки применены");
      return;
    }

    const panelSettingsReset = event.target.closest("[data-panel-settings-reset]");
    if (panelSettingsReset) {
      resetPanelSettings();
      return;
    }

    const clearFilter = event.target.closest("[data-filter-clear]");
    if (clearFilter) {
      const input = clearFilter.parentElement?.querySelector("[data-filter]");
      if (input) {
        input.value = "";
        updateFilter(input);
        input.focus();
      }
      return;
    }

    const aiLogExpand = event.target.closest("[data-ai-log-expand]");
    if (aiLogExpand) {
      setAILogDetails(aiLogExpand.dataset.aiLogExpand === "open");
      return;
    }

    const copy = event.target.closest("[data-copy], [data-copy-target]");
    if (copy) {
      const target = copy.dataset.copyTarget;
      if (target) {
        const id = target.startsWith("#") ? target.slice(1) : target;
        copyValue(document.getElementById(id)?.innerText || "");
      } else {
        copyValue(copy.dataset.copy || copy.innerText);
      }
      return;
    }

    const code = event.target.closest("code");
    if (code) {
      copyValue(code.innerText);
      return;
    }

    const random = event.target.closest("[data-random]");
    if (random) {
      const cards = [...document.querySelectorAll(random.dataset.random)].filter(function (node) {
        return node.style.display !== "none";
      });
      if (!cards.length) return;
      document.querySelectorAll(".picked").forEach(function (node) { node.classList.remove("picked"); });
      const card = cards[Math.floor(Math.random() * cards.length)];
      card.classList.add("picked");
      card.scrollIntoView({behavior: "smooth", block: "center"});
      return;
    }

    const exportButton = event.target.closest("[data-export-csv]");
    if (exportButton) {
      exportTable(exportButton);
      return;
    }

    const sortable = event.target.closest("th[data-sort]");
    if (sortable) {
      sortTable(sortable.closest("table"), Number(sortable.dataset.sort), sortable);
      return;
    }

    maybeNavigateWithTransition(event);
  });

  document.addEventListener("keydown", function (event) {
    const th = event.target.closest?.("th[data-sort]");
    if (th && (event.key === "Enter" || event.key === " ")) {
      event.preventDefault();
      sortTable(th.closest("table"), Number(th.dataset.sort), th);
      return;
    }
    if (event.key === "/" && !event.ctrlKey && !event.metaKey && !event.altKey) {
      const active = document.activeElement;
      if (active && ["INPUT", "TEXTAREA", "SELECT"].includes(active.tagName)) return;
      const filter = document.querySelector("[data-filter]");
      if (filter) {
        event.preventDefault();
        filter.focus();
      }
    }
  });

  document.addEventListener("submit", function (event) {
    const message = event.target.dataset.confirm;
    if (message && !window.confirm(message)) {
      event.preventDefault();
    }
  });

  window.addEventListener("storage", function (event) {
    if (event.key === themeKey) {
      applyTheme(preferredTheme());
      applyPanelSettings(panelSettings, true);
    }
    if (event.key === panelSettingsKey) {
      panelSettings = readPanelSettings();
      pendingPanelSettings = {...panelSettings, colors: {...panelSettings.colors}};
      applyPanelSettings(panelSettings, true);
    }
  });

  document.querySelectorAll("[data-editor]").forEach(updatePreview);
  document.querySelectorAll("[data-filter]").forEach(updateFilter);
  updateAILogFilters();
  enhanceColorPickers();
  fillPanelSettingsControls(pendingPanelSettings);
  enhanceTables();
})();
