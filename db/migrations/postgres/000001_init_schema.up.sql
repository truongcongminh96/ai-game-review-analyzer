-- Lean MVP schema for storing Steam game review analyses.
-- This script is idempotent enough for local/Supabase bootstrap:
-- - creates missing objects
-- - upgrades the main integrity constraints from the first draft
-- - removes redundant indexes

create extension if not exists pgcrypto;

create or replace function public.set_updated_at()
returns trigger
language plpgsql
as $$
begin
  new.updated_at = now();
  return new;
end;
$$;

do $$
begin
  if to_regtype('public.analysis_status') is null then
    create type public.analysis_status as enum (
      'pending',
      'success',
      'failed'
    );
  end if;
end
$$;

create table if not exists public.games (
  id uuid primary key default gen_random_uuid(),
  steam_app_id varchar(50) not null unique,
  title varchar(255) not null,
  cover_url text,
  genre varchar(255),
  release_year integer,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  constraint chk_games_title_not_blank
    check (btrim(title) <> ''),
  constraint chk_games_release_year_range
    check (release_year is null or release_year between 1950 and 2100)
);

drop index if exists public.idx_games_steam_app_id;

create index if not exists idx_games_title
  on public.games (title);

do $$
begin
  if not exists (
    select 1
    from pg_constraint
    where conname = 'chk_games_title_not_blank'
      and conrelid = 'public.games'::regclass
  ) then
    alter table public.games
      add constraint chk_games_title_not_blank
      check (btrim(title) <> '');
  end if;

  if not exists (
    select 1
    from pg_constraint
    where conname = 'chk_games_release_year_range'
      and conrelid = 'public.games'::regclass
  ) then
    alter table public.games
      add constraint chk_games_release_year_range
      check (release_year is null or release_year between 1950 and 2100);
  end if;
end
$$;

drop trigger if exists trg_games_updated_at on public.games;
create trigger trg_games_updated_at
before update on public.games
for each row
execute function public.set_updated_at();

create table if not exists public.analysis_runs (
  id uuid primary key default gen_random_uuid(),
  game_id uuid not null references public.games(id) on delete cascade,
  review_limit integer not null default 30 check (review_limit > 0),
  language varchar(50) not null default 'english',
  genre text,
  review_count integer not null default 0 check (review_count >= 0),
  status public.analysis_status not null default 'pending',
  requested_at timestamptz not null default now(),
  completed_at timestamptz,
  error_message text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  constraint chk_analysis_runs_language_not_blank
    check (btrim(language) <> ''),
  constraint chk_analysis_runs_completed_at_order
    check (completed_at is null or completed_at >= requested_at),
  constraint chk_analysis_runs_status_fields
    check (
      (status = 'pending' and completed_at is null and error_message is null)
      or (status = 'success' and completed_at is not null and error_message is null)
      or (status = 'failed' and completed_at is not null and btrim(coalesce(error_message, '')) <> '')
    )
);

create index if not exists idx_analysis_runs_game_id
  on public.analysis_runs (game_id);

create index if not exists idx_analysis_runs_status
  on public.analysis_runs (status);

create index if not exists idx_analysis_runs_requested_at
  on public.analysis_runs (requested_at desc);

create index if not exists idx_analysis_runs_lookup
  on public.analysis_runs (game_id, language, review_limit, requested_at desc);

do $$
begin
  if not exists (
    select 1
    from pg_constraint
    where conname = 'chk_analysis_runs_language_not_blank'
      and conrelid = 'public.analysis_runs'::regclass
  ) then
    alter table public.analysis_runs
      add constraint chk_analysis_runs_language_not_blank
      check (btrim(language) <> '');
  end if;

  if not exists (
    select 1
    from pg_constraint
    where conname = 'chk_analysis_runs_completed_at_order'
      and conrelid = 'public.analysis_runs'::regclass
  ) then
    alter table public.analysis_runs
      add constraint chk_analysis_runs_completed_at_order
      check (completed_at is null or completed_at >= requested_at);
  end if;

  if not exists (
    select 1
    from pg_constraint
    where conname = 'chk_analysis_runs_status_fields'
      and conrelid = 'public.analysis_runs'::regclass
  ) then
    alter table public.analysis_runs
      add constraint chk_analysis_runs_status_fields
      check (
        (status = 'pending' and completed_at is null and error_message is null)
        or (status = 'success' and completed_at is not null and error_message is null)
        or (status = 'failed' and completed_at is not null and btrim(coalesce(error_message, '')) <> '')
      );
  end if;
end
$$;

drop trigger if exists trg_analysis_runs_updated_at on public.analysis_runs;
create trigger trg_analysis_runs_updated_at
before update on public.analysis_runs
for each row
execute function public.set_updated_at();

create table if not exists public.analysis_results (
  id uuid primary key default gen_random_uuid(),
  analysis_run_id uuid not null unique references public.analysis_runs(id) on delete cascade,
  summary text not null,
  praised_features jsonb not null default '[]'::jsonb,
  common_issues jsonb not null default '[]'::jsonb,
  topics jsonb not null default '[]'::jsonb,
  sentiment_positive integer not null default 0 check (sentiment_positive >= 0),
  sentiment_neutral integer not null default 0 check (sentiment_neutral >= 0),
  sentiment_negative integer not null default 0 check (sentiment_negative >= 0),
  model_name varchar(100),
  raw_ai_response jsonb,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  constraint chk_analysis_results_summary_not_blank
    check (btrim(summary) <> ''),
  constraint chk_analysis_results_praised_features_array
    check (jsonb_typeof(praised_features) = 'array'),
  constraint chk_analysis_results_common_issues_array
    check (jsonb_typeof(common_issues) = 'array'),
  constraint chk_analysis_results_topics_array
    check (jsonb_typeof(topics) = 'array'),
  constraint chk_analysis_results_raw_ai_response_type
    check (raw_ai_response is null or jsonb_typeof(raw_ai_response) in ('object', 'array'))
);

drop index if exists public.idx_analysis_results_analysis_run_id;
drop index if exists public.idx_analysis_results_topics;
drop index if exists public.idx_analysis_results_praised_features;
drop index if exists public.idx_analysis_results_common_issues;

do $$
begin
  if not exists (
    select 1
    from pg_constraint
    where conname = 'chk_analysis_results_summary_not_blank'
      and conrelid = 'public.analysis_results'::regclass
  ) then
    alter table public.analysis_results
      add constraint chk_analysis_results_summary_not_blank
      check (btrim(summary) <> '');
  end if;

  if not exists (
    select 1
    from pg_constraint
    where conname = 'chk_analysis_results_praised_features_array'
      and conrelid = 'public.analysis_results'::regclass
  ) then
    alter table public.analysis_results
      add constraint chk_analysis_results_praised_features_array
      check (jsonb_typeof(praised_features) = 'array');
  end if;

  if not exists (
    select 1
    from pg_constraint
    where conname = 'chk_analysis_results_common_issues_array'
      and conrelid = 'public.analysis_results'::regclass
  ) then
    alter table public.analysis_results
      add constraint chk_analysis_results_common_issues_array
      check (jsonb_typeof(common_issues) = 'array');
  end if;

  if not exists (
    select 1
    from pg_constraint
    where conname = 'chk_analysis_results_topics_array'
      and conrelid = 'public.analysis_results'::regclass
  ) then
    alter table public.analysis_results
      add constraint chk_analysis_results_topics_array
      check (jsonb_typeof(topics) = 'array');
  end if;

  if not exists (
    select 1
    from pg_constraint
    where conname = 'chk_analysis_results_raw_ai_response_type'
      and conrelid = 'public.analysis_results'::regclass
  ) then
    alter table public.analysis_results
      add constraint chk_analysis_results_raw_ai_response_type
      check (raw_ai_response is null or jsonb_typeof(raw_ai_response) in ('object', 'array'));
  end if;
end
$$;

drop trigger if exists trg_analysis_results_updated_at on public.analysis_results;
create trigger trg_analysis_results_updated_at
before update on public.analysis_results
for each row
execute function public.set_updated_at();
