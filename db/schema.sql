create schema if not exists groupscholar_ops_logbook;

create table if not exists groupscholar_ops_logbook.events (
  id bigserial primary key,
  occurred_at timestamptz not null,
  title text not null,
  category text not null,
  severity text not null,
  owner text not null,
  status text not null,
  notes text not null default '',
  created_at timestamptz not null default now()
);

create index if not exists events_occurred_at_idx
  on groupscholar_ops_logbook.events (occurred_at desc);

create index if not exists events_status_idx
  on groupscholar_ops_logbook.events (status);

create index if not exists events_category_idx
  on groupscholar_ops_logbook.events (category);
