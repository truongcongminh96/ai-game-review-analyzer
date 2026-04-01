drop table if exists public.analysis_item_evidence;
drop table if exists public.analysis_insight_items;
drop table if exists public.review_snapshots;

alter table public.analysis_runs
  drop column if exists started_at,
  drop column if exists progress_percent,
  drop column if exists current_stage;
