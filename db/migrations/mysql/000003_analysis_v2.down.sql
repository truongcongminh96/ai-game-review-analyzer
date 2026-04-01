drop table if exists analysis_item_evidence;
drop table if exists analysis_insight_items;
drop table if exists review_snapshots;

alter table analysis_runs
  drop column started_at,
  drop column progress_percent,
  drop column current_stage;
