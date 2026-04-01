alter table public.analysis_runs
  add column if not exists current_stage varchar(50) not null default 'queued',
  add column if not exists progress_percent integer not null default 0,
  add column if not exists started_at timestamptz;

create table if not exists public.review_snapshots (
  id uuid primary key default gen_random_uuid(),
  analysis_run_id uuid not null references public.analysis_runs(id) on delete cascade,
  source varchar(20) not null default 'steam',
  source_review_id varchar(100),
  review_index integer not null,
  review_text text not null,
  voted_up boolean not null,
  language varchar(50) not null,
  helpful_votes integer not null default 0,
  funny_votes integer not null default 0,
  weighted_vote_score numeric(8,6) not null default 0,
  steam_created_at timestamptz,
  playtime_forever_min integer not null default 0,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (analysis_run_id, review_index)
);

create index if not exists idx_review_snapshots_run_id
  on public.review_snapshots (analysis_run_id, review_index);

drop trigger if exists trg_review_snapshots_updated_at on public.review_snapshots;
create trigger trg_review_snapshots_updated_at
before update on public.review_snapshots
for each row
execute function public.set_updated_at();

create table if not exists public.analysis_insight_items (
  id uuid primary key default gen_random_uuid(),
  analysis_run_id uuid not null references public.analysis_runs(id) on delete cascade,
  kind varchar(20) not null,
  label varchar(120) not null,
  summary text not null,
  severity integer,
  confidence numeric(4,3) not null default 0.5,
  evidence_count integer not null default 0,
  sort_order integer not null default 0,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (analysis_run_id, kind, label)
);

create index if not exists idx_analysis_insight_items_run_kind
  on public.analysis_insight_items (analysis_run_id, kind, sort_order);

drop trigger if exists trg_analysis_insight_items_updated_at on public.analysis_insight_items;
create trigger trg_analysis_insight_items_updated_at
before update on public.analysis_insight_items
for each row
execute function public.set_updated_at();

create table if not exists public.analysis_item_evidence (
  id uuid primary key default gen_random_uuid(),
  insight_item_id uuid not null references public.analysis_insight_items(id) on delete cascade,
  review_snapshot_id uuid not null references public.review_snapshots(id) on delete cascade,
  quote text not null,
  confidence numeric(4,3) not null default 0.5,
  created_at timestamptz not null default now(),
  unique (insight_item_id, review_snapshot_id)
);

create index if not exists idx_analysis_item_evidence_item
  on public.analysis_item_evidence (insight_item_id);
