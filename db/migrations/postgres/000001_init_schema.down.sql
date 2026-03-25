drop trigger if exists trg_analysis_results_updated_at on public.analysis_results;
drop trigger if exists trg_analysis_runs_updated_at on public.analysis_runs;
drop trigger if exists trg_games_updated_at on public.games;

drop table if exists public.analysis_results;
drop table if exists public.analysis_runs;
drop table if exists public.games;

drop function if exists public.set_updated_at();

do $$
begin
  if to_regtype('public.analysis_status') is not null then
    drop type public.analysis_status;
  end if;
end
$$;
