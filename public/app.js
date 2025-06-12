const form = document.getElementById("event-form");
const list = document.getElementById("event-list");
const statusFilter = document.getElementById("status-filter");
const categoryFilter = document.getElementById("category-filter");
const refreshBtn = document.getElementById("refresh");
const summaryGrid = document.getElementById("summary-grid");
const summaryCategory = document.getElementById("summary-category");
const summaryOwner = document.getElementById("summary-owner");
const summaryLatest = document.getElementById("summary-latest");

const toRFC3339 = (value) => {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return date.toISOString();
};

const badgeClass = (severity) => {
  const key = severity.toLowerCase();
  if (key === "high") return "badge high";
  if (key === "medium") return "badge medium";
  return "badge low";
};

const formatDate = (iso) => {
  const date = new Date(iso);
  return date.toLocaleString("en-US", {
    month: "short",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
};

const formatDateLong = (iso) => {
  if (!iso) return "—";
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return "—";
  return date.toLocaleString("en-US", {
    month: "short",
    day: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
};

const createCell = (text) => {
  const cell = document.createElement("span");
  cell.textContent = text;
  return cell;
};

const renderEvents = (events) => {
  list.innerHTML = "";

  if (!events.length) {
    const empty = document.createElement("div");
    empty.className = "empty";
    empty.textContent = "No signals yet. Log the first one.";
    list.appendChild(empty);
    return;
  }

  events.forEach((event) => {
    const item = document.createElement("div");
    item.className = "event-item";

    const row = document.createElement("div");
    row.className = "row";
    row.appendChild(createCell(formatDate(event.occurred_at)));
    row.appendChild(createCell(event.title));
    row.appendChild(createCell(event.category));

    const severityCell = document.createElement("span");
    const severityBadge = document.createElement("span");
    severityBadge.className = badgeClass(event.severity);
    severityBadge.textContent = event.severity;
    severityCell.appendChild(severityBadge);
    row.appendChild(severityCell);

    row.appendChild(createCell(event.owner));
    row.appendChild(createCell(event.status));

    const details = document.createElement("div");
    details.className = "row-details";

    const notesBlock = document.createElement("div");
    notesBlock.className = "detail-block";
    const notesTitle = document.createElement("strong");
    notesTitle.textContent = "Notes";
    const notesText = document.createElement("p");
    notesText.textContent = event.notes || "No notes captured.";
    notesBlock.appendChild(notesTitle);
    notesBlock.appendChild(notesText);

    const metaBlock = document.createElement("div");
    metaBlock.className = "detail-meta";
    metaBlock.appendChild(createCell(`Occurred: ${formatDateLong(event.occurred_at)}`));
    metaBlock.appendChild(createCell(`Logged: ${formatDateLong(event.created_at)}`));

    details.appendChild(notesBlock);
    details.appendChild(metaBlock);

    row.addEventListener("click", () => {
      item.classList.toggle("open");
    });

    item.appendChild(row);
    item.appendChild(details);
    list.appendChild(item);
  });
};

const renderSummary = (summary) => {
  summaryGrid.innerHTML = "";

  const cards = [
    { label: "Total signals", value: summary.total_count },
    { label: "Open", value: summary.open_count },
    { label: "Monitoring", value: summary.monitoring_count },
    { label: "Resolved", value: summary.resolved_count },
    { label: "High severity", value: summary.high_count },
    { label: "Medium severity", value: summary.medium_count },
    { label: "Low severity", value: summary.low_count },
  ];

  cards.forEach((card) => {
    const node = document.createElement("div");
    node.className = "summary-card";

    const label = document.createElement("div");
    label.className = "summary-label";
    label.textContent = card.label;

    const value = document.createElement("div");
    value.className = "summary-value";
    value.textContent = card.value ?? 0;

    node.appendChild(label);
    node.appendChild(value);
    summaryGrid.appendChild(node);
  });

  summaryCategory.textContent = summary.top_category
    ? `${summary.top_category} (${summary.top_category_count})`
    : "—";
  summaryOwner.textContent = summary.top_owner
    ? `${summary.top_owner} (${summary.top_owner_count})`
    : "—";
  summaryLatest.textContent = summary.latest_occurred
    ? formatDateLong(summary.latest_occurred)
    : "—";
};

const buildParams = () => {
  const params = new URLSearchParams();
  if (statusFilter.value) params.append("status", statusFilter.value);
  if (categoryFilter.value) params.append("category", categoryFilter.value.trim());
  return params;
};

const loadEvents = async () => {
  const params = buildParams();
  const response = await fetch(`/api/events?${params.toString()}`);
  if (!response.ok) return;
  const data = await response.json();
  renderEvents(data);
};

const loadSummary = async () => {
  const params = buildParams();
  params.append("view", "summary");
  const response = await fetch(`/api/events?${params.toString()}`);
  if (!response.ok) return;
  const data = await response.json();
  renderSummary(data);
};

const refreshData = async () => {
  await Promise.all([loadEvents(), loadSummary()]);
};

form.addEventListener("submit", async (event) => {
  event.preventDefault();
  const formData = new FormData(form);
  const payload = {
    title: formData.get("title"),
    category: formData.get("category"),
    severity: formData.get("severity"),
    owner: formData.get("owner"),
    status: formData.get("status"),
    notes: formData.get("notes"),
    occurred_at: toRFC3339(formData.get("occurred_at")),
  };

  const response = await fetch("/api/events", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    alert("Could not save the entry. Please try again.");
    return;
  }

  form.reset();
  await refreshData();
});

refreshBtn.addEventListener("click", refreshData);
statusFilter.addEventListener("change", refreshData);
categoryFilter.addEventListener("input", () => {
  window.clearTimeout(categoryFilter._debounce);
  categoryFilter._debounce = window.setTimeout(refreshData, 400);
});

refreshData();
