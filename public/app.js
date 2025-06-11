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
    const row = document.createElement("div");
    row.className = "row";
    row.innerHTML = `
      <span>${formatDate(event.occurred_at)}</span>
      <span>${event.title}</span>
      <span>${event.category}</span>
      <span><span class="${badgeClass(event.severity)}">${event.severity}</span></span>
      <span>${event.owner}</span>
      <span>${event.status}</span>
    `;
    list.appendChild(row);
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
    node.innerHTML = `
      <div class="summary-label">${card.label}</div>
      <div class="summary-value">${card.value}</div>
    `;
    summaryGrid.appendChild(node);
  });

  summaryCategory.textContent = summary.top_category
    ? `${summary.top_category} (${summary.top_category_count})`
    : "—";
  summaryOwner.textContent = summary.top_owner
    ? `${summary.top_owner} (${summary.top_owner_count})`
    : "—";
  summaryLatest.textContent = summary.latest_occurred
    ? formatDate(summary.latest_occurred)
    : "—";
};

const loadEvents = async () => {
  const params = new URLSearchParams();
  if (statusFilter.value) params.append("status", statusFilter.value);
  if (categoryFilter.value) params.append("category", categoryFilter.value.trim());
  const response = await fetch(`/api/events?${params.toString()}`);
  if (!response.ok) return;
  const data = await response.json();
  renderEvents(data);
};

const loadSummary = async () => {
  const params = new URLSearchParams();
  params.append("view", "summary");
  if (statusFilter.value) params.append("status", statusFilter.value);
  if (categoryFilter.value) params.append("category", categoryFilter.value.trim());
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
