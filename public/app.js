const form = document.getElementById("event-form");
const list = document.getElementById("event-list");
const statusFilter = document.getElementById("status-filter");
const categoryFilter = document.getElementById("category-filter");
const refreshBtn = document.getElementById("refresh");

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

const loadEvents = async () => {
  const params = new URLSearchParams();
  if (statusFilter.value) params.append("status", statusFilter.value);
  if (categoryFilter.value) params.append("category", categoryFilter.value.trim());
  const response = await fetch(`/api/events?${params.toString()}`);
  if (!response.ok) return;
  const data = await response.json();
  renderEvents(data);
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
  await loadEvents();
});

refreshBtn.addEventListener("click", loadEvents);
statusFilter.addEventListener("change", loadEvents);
categoryFilter.addEventListener("input", () => {
  window.clearTimeout(categoryFilter._debounce);
  categoryFilter._debounce = window.setTimeout(loadEvents, 400);
});

loadEvents();
