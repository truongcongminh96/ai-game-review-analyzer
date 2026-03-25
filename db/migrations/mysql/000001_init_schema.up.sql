create table if not exists games (
  id char(36) not null primary key,
  steam_app_id varchar(50) not null,
  title varchar(255) not null,
  cover_url text null,
  genre varchar(255) null,
  release_year int null,
  created_at datetime(6) not null default current_timestamp(6),
  updated_at datetime(6) not null default current_timestamp(6) on update current_timestamp(6),
  constraint uq_games_steam_app_id unique (steam_app_id),
  constraint chk_games_title_not_blank check (trim(title) <> ''),
  constraint chk_games_release_year_range check (release_year is null or release_year between 1950 and 2100)
) engine=InnoDB default charset=utf8mb4 collate=utf8mb4_unicode_ci;

create index idx_games_title
  on games (title);

create table if not exists analysis_runs (
  id char(36) not null primary key,
  game_id char(36) not null,
  review_limit int not null default 30,
  language varchar(50) not null default 'english',
  genre text null,
  review_count int not null default 0,
  status enum('pending', 'success', 'failed') not null default 'pending',
  requested_at datetime(6) not null default current_timestamp(6),
  completed_at datetime(6) null,
  error_message text null,
  created_at datetime(6) not null default current_timestamp(6),
  updated_at datetime(6) not null default current_timestamp(6) on update current_timestamp(6),
  constraint fk_analysis_runs_game_id
    foreign key (game_id) references games(id) on delete cascade,
  constraint chk_analysis_runs_review_limit check (review_limit > 0),
  constraint chk_analysis_runs_review_count check (review_count >= 0),
  constraint chk_analysis_runs_language_not_blank check (trim(language) <> ''),
  constraint chk_analysis_runs_completed_at_order check (completed_at is null or completed_at >= requested_at),
  constraint chk_analysis_runs_status_fields check (
    (status = 'pending' and completed_at is null and error_message is null)
    or (status = 'success' and completed_at is not null and error_message is null)
    or (status = 'failed' and completed_at is not null and trim(coalesce(error_message, '')) <> '')
  )
) engine=InnoDB default charset=utf8mb4 collate=utf8mb4_unicode_ci;

create index idx_analysis_runs_game_id
  on analysis_runs (game_id);

create index idx_analysis_runs_status
  on analysis_runs (status);

create index idx_analysis_runs_requested_at
  on analysis_runs (requested_at);

create index idx_analysis_runs_lookup
  on analysis_runs (game_id, language, review_limit, requested_at);

create table if not exists analysis_results (
  id char(36) not null primary key,
  analysis_run_id char(36) not null,
  summary text not null,
  praised_features json not null,
  common_issues json not null,
  topics json not null,
  sentiment_positive int not null default 0,
  sentiment_neutral int not null default 0,
  sentiment_negative int not null default 0,
  model_name varchar(100) null,
  raw_ai_response json null,
  created_at datetime(6) not null default current_timestamp(6),
  updated_at datetime(6) not null default current_timestamp(6) on update current_timestamp(6),
  constraint uq_analysis_results_analysis_run_id unique (analysis_run_id),
  constraint fk_analysis_results_analysis_run_id
    foreign key (analysis_run_id) references analysis_runs(id) on delete cascade,
  constraint chk_analysis_results_summary_not_blank check (trim(summary) <> ''),
  constraint chk_analysis_results_sentiment_positive check (sentiment_positive >= 0),
  constraint chk_analysis_results_sentiment_neutral check (sentiment_neutral >= 0),
  constraint chk_analysis_results_sentiment_negative check (sentiment_negative >= 0)
) engine=InnoDB default charset=utf8mb4 collate=utf8mb4_unicode_ci;
