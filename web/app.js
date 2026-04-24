const EMPTY_PROJECT = "__empty_project__";

const state = {
  config: null,
  trace: null,
  selectedProjects: new Set(),
  selectedServices: new Set(),
  selectedLevels: new Set(),
};

const EVENT_TYPE_LABELS = {
  chat_finished: "Чат завершён",
  delivered: "Доставлено",
  error: "Ошибка",
  info: "Информация",
  nlu_request: "Запрос в NLU",
  nlu_response: "Ответ NLU",
  operator_assigned: "Оператор назначен",
  operator_returned: "Возврат к боту",
  operator_unassigned: "Оператор снят",
  processing_started: "Начата обработка",
  published: "Опубликовано",
  received: "Получено",
  skipped: "Пропущено",
  thrown_away: "Отброшено",
  timeout_removed: "Таймаут снят",
  timeout_sent: "Таймаут отправлен",
  transformed: "Преобразовано",
  unknown: "Не классифицировано",
  warn: "Предупреждение",
};

const REASON_LABELS = {
  chat_event_skipped: "Событие чата пропущено",
  invalid_json: "Некорректный JSON",
  invalid_subject_format: "Некорректный формат subject",
  message_thrown_away: "Сообщение отброшено",
  not_assigned_chat: "Чат не был назначен оператору",
  removed_timeout: "Таймаут снят",
  session_cleared: "Сессия очищена",
  timeout_spam: "Сессия завершена из-за спама таймаутов",
  unsupported_subject: "Неподдерживаемый subject",
};

const FINAL_STATE_LABELS = {
  no_data: "Данные не найдены",
};

const MESSAGE_KIND_LABELS = {
  command: "Командное",
  content: "Переписка",
  system: "Системное",
};

const LEVEL_LABELS = {
  debug: "Отладка",
  error: "Ошибки",
  info: "Инфо",
  warn: "Предупреждения",
};

async function loadConfig() {
  const response = await fetch("/api/config");
  const config = await response.json();
  state.config = config;
  renderSource(config);
}

function renderSource(config) {
  const meta = document.querySelector(".topbar-meta");
  const badge = document.getElementById("mode-badge");
  const hint = document.getElementById("source-hint");

  if (!config.victorialogs_configured) {
    meta.classList.add("hidden");
    return;
  }

  meta.classList.remove("hidden");
  badge.textContent = "VictoriaLogs";

  const parts = [];
  if (config.victorialogs_base_url) {
    parts.push(`адрес ${config.victorialogs_base_url}`);
  }
  if (config.victorialogs_account_id) {
    parts.push(`AccountID ${config.victorialogs_account_id}`);
  }
  if (config.victorialogs_project_id) {
    parts.push(`ProjectID ${config.victorialogs_project_id}`);
  }
  hint.textContent = parts.length ? `Источник: ${parts.join(" · ")}` : "Источник: VictoriaLogs";
}

function queryParams() {
  const params = new URLSearchParams();
  ["user_id", "session_id", "project", "client", "from", "to"].forEach((id) => {
    const value = document.getElementById(id).value;
    if (value) {
      if (id === "from" || id === "to") {
        params.set(id, new Date(value).toISOString());
      } else {
        params.set(id, value);
      }
    }
  });
  ["hide_debug", "only_anomalies", "collapse_duplicates"].forEach((id) => {
    if (document.getElementById(id).checked) {
      params.set(id, "1");
    }
  });
  return params;
}

function resetForm() {
  document.getElementById("user_id").value = "";
  document.getElementById("session_id").value = "";
  document.getElementById("project").value = "";
  document.getElementById("client").value = "";
  document.getElementById("only_anomalies").checked = false;
  document.getElementById("hide_debug").checked = true;
  document.getElementById("collapse_duplicates").checked = true;
  document.getElementById("from").value = "";
  document.getElementById("to").value = "";
  document.getElementById("details").textContent = "Выберите событие слева";
}

async function search() {
  if (!state.config?.victorialogs_configured) {
    state.trace = null;
    showStatus("VictoriaLogs не настроена, запрос сейчас не выполнить.", "warn");
    hideTrace();
    return;
  }

  const sessionID = document.getElementById("session_id").value.trim();
  if (!sessionID) {
    state.trace = null;
    showStatus("Укажи session id, без него точную трассу не собрать.", "warn");
    hideTrace();
    return;
  }

  const params = queryParams();
  history.replaceState(null, "", `/?${params.toString()}`);
  showStatus("Ищу события в VictoriaLogs...", "info");

  try {
    const response = await fetch(`/api/chat-trace?${params.toString()}`);
    const data = await response.json();
    if (!response.ok) {
      state.trace = null;
      showStatus(translateError(data.error) || "Не получилось собрать трассу из VictoriaLogs.", "error");
      hideTrace();
      return;
    }

    state.trace = data;
    syncSelectedFilters();
    if (data.summary?.limit_reached) {
      showStatus(
        `VictoriaLogs вернула ${data.raw_count} строк. Упёрлись в текущий лимит, так что история может быть неполной. Лучше сузить запрос по session id, project или client.`,
        "warn"
      );
    } else {
      hideStatus();
    }
    renderFiltersPanel();
    render();
  } catch (error) {
    state.trace = null;
    showStatus(`Ошибка запроса к VictoriaLogs: ${error.message}`, "error");
    hideTrace();
  }
}

function syncSelectedFilters() {
  state.selectedProjects = new Set(availableProjects());
  state.selectedServices = new Set(availableServices());
  state.selectedLevels = new Set(availableLevels());
}

function projectFilterValue(event) {
  return event?.chat?.project || EMPTY_PROJECT;
}

function projectFilterLabel(value) {
  return value === EMPTY_PROJECT ? "Без проекта" : value;
}

function availableProjects() {
  if (!state.trace) {
    return [];
  }

  const all = new Set();
  [...state.trace.timeline, ...state.trace.anomalies].forEach((event) => {
    all.add(projectFilterValue(event));
  });

  return Array.from(all).sort((left, right) =>
    projectFilterLabel(left).localeCompare(projectFilterLabel(right), "ru")
  );
}

function availableServices() {
  if (!state.trace) {
    return [];
  }

  const all = new Set();
  [...state.trace.timeline, ...state.trace.anomalies].forEach((event) => {
    if (event.service) {
      all.add(event.service);
    }
  });

  return Array.from(all).sort((left, right) => left.localeCompare(right, "ru"));
}

function availableLevels() {
  if (!state.trace) {
    return [];
  }

  const all = new Set();
  [...state.trace.timeline, ...state.trace.anomalies].forEach((event) => {
    if (event.level) {
      all.add(event.level);
    }
  });

  return Array.from(all).sort((left, right) => left.localeCompare(right, "ru"));
}

function renderFilterGroup(kind, title, values, selectedSet, formatter) {
  return `
    <section class="filter-card">
      <div class="filter-card-head">
        <h3>${escapeHTML(title)}</h3>
        <div class="filter-actions">
          <button type="button" class="secondary" data-action="all" data-kind="${escapeHTML(kind)}">Все</button>
          <button type="button" class="secondary" data-action="none" data-kind="${escapeHTML(kind)}">Снять все</button>
        </div>
      </div>
      <div class="filter-list">
        ${
          values.length
            ? values
                .map(
                  (value) => `
                    <label class="filter-chip">
                      <input
                        type="checkbox"
                        data-kind="${escapeHTML(kind)}"
                        value="${escapeHTML(value)}"
                        ${selectedSet.has(value) ? "checked" : ""}
                      />
                      <span>${escapeHTML(formatter(value))}</span>
                    </label>
                  `
                )
                .join("")
            : '<div class="filter-empty">Нет данных</div>'
        }
      </div>
    </section>
  `;
}

function buildFiltersCaption(projects, services, levels) {
  const parts = [
    `Проекты ${state.selectedProjects.size}/${projects.length}`,
    `Уровни ${state.selectedLevels.size}/${levels.length}`,
    `Сервисы ${state.selectedServices.size}/${services.length}`,
  ];
  return parts.join(" · ");
}

function bindFilterCheckboxes() {
  const body = document.getElementById("filters-body");

  body.querySelectorAll('input[type="checkbox"][data-kind]').forEach((input) => {
    input.addEventListener("change", () => {
      let targetSet = state.selectedServices;
      if (input.dataset.kind === "project") {
        targetSet = state.selectedProjects;
      } else if (input.dataset.kind === "level") {
        targetSet = state.selectedLevels;
      }

      if (input.checked) {
        targetSet.add(input.value);
      } else {
        targetSet.delete(input.value);
      }

      renderFiltersPanel();
      render();
    });
  });

  body.querySelectorAll("button[data-action][data-kind]").forEach((button) => {
    button.addEventListener("click", () => {
      const kind = button.dataset.kind;
      const action = button.dataset.action;

      let values = [];
      let targetSet;

      if (kind === "project") {
        values = availableProjects();
        targetSet = state.selectedProjects;
      } else if (kind === "level") {
        values = availableLevels();
        targetSet = state.selectedLevels;
      } else {
        values = availableServices();
        targetSet = state.selectedServices;
      }

      targetSet.clear();
      if (action === "all") {
        values.forEach((value) => targetSet.add(value));
      }

      renderFiltersPanel();
      render();
    });
  });
}

function renderFiltersPanel() {
  const panel = document.getElementById("filters-panel");
  const body = document.getElementById("filters-body");
  const caption = document.getElementById("filters-caption");

  const projects = availableProjects();
  const services = availableServices();
  const levels = availableLevels();

  if (!projects.length && !services.length && !levels.length) {
    panel.classList.add("hidden");
    body.innerHTML = "";
    caption.textContent = "";
    return;
  }

  panel.classList.remove("hidden");
  caption.textContent = buildFiltersCaption(projects, services, levels);

  body.innerHTML = `
    <div class="filter-groups">
      ${renderFilterGroup("project", "Проекты", projects, state.selectedProjects, projectFilterLabel)}
      ${renderFilterGroup("level", "Уровни логов", levels, state.selectedLevels, translateLevel)}
      ${renderFilterGroup("service", "Сервисы", services, state.selectedServices, (value) => value)}
    </div>
  `;

  bindFilterCheckboxes();
}

function render() {
  if (!state.trace) {
    hideTrace();
    return;
  }

  document.getElementById("summary").classList.remove("hidden");
  document.getElementById("results").classList.remove("hidden");

  renderSummary();
  renderEvents("timeline", visibleTimeline());
  renderEvents("anomalies", visibleAnomalies());

  const details = document.getElementById("details");
  if (!details.textContent || details.textContent === "Выберите событие слева") {
    details.textContent = "Выберите событие слева";
  }
}

function renderSummary() {
  const timeline = visibleTimeline();
  const services = Array.from(
    new Set(
      timeline
        .map((event) => event.service)
        .filter(Boolean)
    )
  ).sort((left, right) => left.localeCompare(right, "ru"));

  const lastEvent = timeline[timeline.length - 1];

  document.getElementById("summary").innerHTML = [
    summaryItem("Чат", state.trace.summary.chat_key),
    summaryItem("События", timeline.length),
    summaryItem("Ошибки", countEvents(timeline, isErrorEvent), "error"),
    summaryItem("Предупреждения", countEvents(timeline, isWarnEvent), "warn"),
    summaryItem("Пропуски", countEvents(timeline, (event) => event.event_type === "skipped")),
    summaryItem("Финальное состояние", lastEvent ? translateFinalState(finalStateFromEvent(lastEvent)) : "Данные не найдены"),
    summaryItem("Сервисы", services.length ? services.join(", ") : "—"),
  ].join("");
}

function renderEvents(targetId, events) {
  const target = document.getElementById(targetId);
  if (!events.length) {
    target.innerHTML = '<div class="empty">В текущем фильтре событий не найдено</div>';
    return;
  }

  target.innerHTML = events
    .map(
      (event, index) => `
      <button class="event ${escapeHTML(event.level || "")}" data-target="${escapeHTML(targetId)}" data-index="${index}">
        <span class="event-top">
          <span class="time">${escapeHTML(formatTimestamp(event.timestamp))}</span>
          <span class="pill">${escapeHTML(event.service || "неизвестно")}</span>
          <span class="pill subtle">${escapeHTML(translateEventType(event.event_type))}</span>
          <span class="pill kind">${escapeHTML(translateMessageKind(event.message_kind))}</span>
          ${
            event.from || event.to
              ? `<span class="pill route">${escapeHTML(formatRoute(event.from, event.to))}</span>`
              : ""
          }
        </span>
        <span class="desc">${escapeHTML(event.description || "Без описания")}</span>
        ${
          event.reason
            ? `<span class="reason">Причина: ${escapeHTML(translateReason(event.reason))}</span>`
            : ""
        }
      </button>`
    )
    .join("");

  target.querySelectorAll(".event").forEach((button) => {
    button.addEventListener("click", () => {
      const list = button.dataset.target === "timeline" ? visibleTimeline() : visibleAnomalies();
      const event = list[Number(button.dataset.index)];
      document.getElementById("details").textContent = JSON.stringify(event, null, 2);
    });
  });
}

function visibleTimeline() {
  return filterVisibleEvents(state.trace?.timeline || []);
}

function visibleAnomalies() {
  return filterVisibleEvents(state.trace?.anomalies || []);
}

function filterVisibleEvents(events) {
  if (!events.length) {
    return [];
  }

  return events.filter((event) => {
    const project = projectFilterValue(event);
    return (
      state.selectedProjects.has(project) &&
      state.selectedServices.has(event.service) &&
      state.selectedLevels.has(event.level)
    );
  });
}

function countEvents(events, predicate) {
  return events.reduce((count, event) => count + (predicate(event) ? 1 : 0), 0);
}

function isErrorEvent(event) {
  return event.event_type === "error" || event.event_type === "thrown_away" || event.level === "error";
}

function isWarnEvent(event) {
  return event.event_type === "warn" || event.level === "warn";
}

function finalStateFromEvent(event) {
  if (event.reason) {
    return `${event.event_type}:${event.reason}`;
  }
  return event.event_type;
}

function summaryItem(label, value, tone = "") {
  return `<div class="summary-item ${tone}"><span>${escapeHTML(label)}</span><strong>${escapeHTML(formatValue(value))}</strong></div>`;
}

function setupTabs() {
  document.querySelectorAll(".tab").forEach((tab) => {
    tab.addEventListener("click", () => {
      document.querySelectorAll(".tab").forEach((button) => button.classList.remove("active"));
      document.querySelectorAll(".tab-panel").forEach((panel) => panel.classList.remove("active"));
      tab.classList.add("active");
      document.getElementById(tab.dataset.tab).classList.add("active");
    });
  });
}

function showStatus(message, tone = "info") {
  const status = document.getElementById("status");
  status.className = `status ${tone}`;
  status.textContent = message;
}

function hideStatus() {
  const status = document.getElementById("status");
  status.className = "status hidden";
  status.textContent = "";
}

function hideTrace() {
  document.getElementById("summary").classList.add("hidden");
  document.getElementById("results").classList.add("hidden");
  document.getElementById("filters-panel").classList.add("hidden");
  document.getElementById("filters-body").innerHTML = "";
  document.getElementById("filters-caption").textContent = "";
  document.getElementById("summary").innerHTML = "";
  document.getElementById("timeline").innerHTML = "";
  document.getElementById("anomalies").innerHTML = "";
  document.getElementById("details").textContent = "Выберите событие слева";
}

function translateEventType(value) {
  return EVENT_TYPE_LABELS[value] || value || "Неизвестно";
}

function translateReason(value) {
  return REASON_LABELS[value] || value || "Не указана";
}

function translateFinalState(value) {
  if (!value) {
    return "—";
  }

  if (FINAL_STATE_LABELS[value]) {
    return FINAL_STATE_LABELS[value];
  }

  const parts = String(value).split(":");
  if (parts.length !== 2) {
    return value;
  }

  const [eventType, reason] = parts;
  return `${translateEventType(eventType)}: ${translateReason(reason)}`;
}

function translateMessageKind(value) {
  return MESSAGE_KIND_LABELS[value] || MESSAGE_KIND_LABELS.system;
}

function formatRoute(from, to) {
  const left = from || "неизвестно";
  const right = to || "неизвестно";
  return `${left} → ${right}`;
}

function translateLevel(value) {
  return LEVEL_LABELS[value] || value || "Неизвестно";
}

function translateError(value) {
  if (!value) {
    return "";
  }
  if (value === "session_id is required for exact chat trace search") {
    return "Для точной трассировки обязательно укажите session id.";
  }
  if (value === "victorialogs is not configured") {
    return "VictoriaLogs не настроена.";
  }
  return value;
}

function formatTimestamp(value) {
  if (!value) {
    return "Без времени";
  }
  return new Date(value).toLocaleString("ru-RU");
}

function formatValue(value) {
  if (value === null || value === undefined || value === "") {
    return "—";
  }
  return String(value);
}

function escapeHTML(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

document.getElementById("search").addEventListener("click", search);
document.getElementById("reset").addEventListener("click", () => {
  resetForm();
  if (state.trace) {
    syncSelectedFilters();
    renderFiltersPanel();
    render();
  }
});

setupTabs();
loadConfig().then(() => {
  if (state.config?.victorialogs_configured) {
    showStatus("Укажи session id и нажми «Показать трассу».", "info");
    hideTrace();
  } else {
    showStatus("VictoriaLogs не настроена. Задай URL, и тогда можно будет искать трассу.", "warn");
    hideTrace();
  }
});
