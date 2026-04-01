alter table analysis_runs
  add column current_stage varchar(50) not null default 'queued',
  add column progress_percent int not null default 0,
  add column started_at datetime(6) null;

create table if not exists review_snapshots (
  id char(36) not null primary key,
  analysis_run_id char(36) not null,
  source varchar(20) not null default 'steam',
  source_review_id varchar(100) null,
  review_index int not null,
  review_text text not null,
  voted_up boolean not null,
  language varchar(50) not null,
  helpful_votes int not null default 0,
  funny_votes int not null default 0,
  weighted_vote_score decimal(8,6) not null default 0,
  steam_created_at datetime(6) null,
  playtime_forever_min int not null default 0,
  created_at datetime(6) not null default current_timestamp(6),
  updated_at datetime(6) not null default current_timestamp(6) on update current_timestamp(6),
  constraint fk_review_snapshots_run
    foreign key (analysis_run_id) references analysis_runs(id) on delete cascade,
  constraint uq_review_snapshots_run_review_index unique (analysis_run_id, review_index)
) engine=InnoDB default charset=utf8mb4 collate=utf8mb4_unicode_ci;

create index idx_review_snapshots_run_id
  on review_snapshots (analysis_run_id, review_index);

create table if not exists analysis_insight_items (
  id char(36) not null primary key,
  analysis_run_id char(36) not null,
  kind varchar(20) not null,
  label varchar(120) not null,
  summary text not null,
  severity int null,
  confidence decimal(4,3) not null default 0.5,
  evidence_count int not null default 0,
  sort_order int not null default 0,
  created_at datetime(6) not null default current_timestamp(6),
  updated_at datetime(6) not null default current_timestamp(6) on update current_timestamp(6),
  constraint fk_analysis_insight_items_run
    foreign key (analysis_run_id) references analysis_runs(id) on delete cascade,
  constraint uq_analysis_insight_items_run_kind_label unique (analysis_run_id, kind, label)
) engine=InnoDB default charset=utf8mb4 collate=utf8mb4_unicode_ci;

create index idx_analysis_insight_items_run_kind
  on analysis_insight_items (analysis_run_id, kind, sort_order);

create table if not exists analysis_item_evidence (
  id char(36) not null primary key,
  insight_item_id char(36) not null,
  review_snapshot_id char(36) not null,
  quote text not null,
  confidence decimal(4,3) not null default 0.5,
  created_at datetime(6) not null default current_timestamp(6),
  constraint fk_analysis_item_evidence_item
    foreign key (insight_item_id) references analysis_insight_items(id) on delete cascade,
  constraint fk_analysis_item_evidence_review
    foreign key (review_snapshot_id) references review_snapshots(id) on delete cascade,
  constraint uq_analysis_item_evidence_item_review unique (insight_item_id, review_snapshot_id)
) engine=InnoDB default charset=utf8mb4 collate=utf8mb4_unicode_ci;

create index idx_analysis_item_evidence_item
  on analysis_item_evidence (insight_item_id);
