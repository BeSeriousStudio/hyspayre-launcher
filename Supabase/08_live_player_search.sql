-- HYSPAYRE · 08 · Live-Spielersuche mit Teilnamen und Seitenweise-Laden.
-- NACH 07_social_profiles_friends.sql einmal im Supabase SQL Editor ausführen.
-- Ändert keine Accounts, Freundschaften oder Rechte bestehender Funktionen.
-- Nur authentifizierte Spieler; KEINE privaten E-Mail-Adressen oder Guthaben.

create or replace function public.bss_social_search_live(
  p_query text, p_offset integer default 0, p_limit integer default 40
)
returns jsonb
language plpgsql stable security definer set search_path = ''
as $$
declare
  me uuid := auth.uid();
  term text := lower(btrim(coalesce(p_query,'')));
  start_at integer := greatest(coalesce(p_offset,0),0);
  page_size integer := least(greatest(coalesce(p_limit,40),1),100);
  total_count integer := 0;
  result_players jsonb := '[]'::jsonb;
begin
  if me is null then raise exception 'Anmeldung erforderlich'; end if;
  if char_length(term) < 1 or char_length(term) > 30 then
    raise exception 'Bitte 1 bis 30 Zeichen für die Suche eingeben';
  end if;

  select count(*) into total_count
  from public.bss_profiles p
  where p.user_id <> me
    and position(term in lower(p.profile_name)) > 0;

  select coalesce(jsonb_agg(jsonb_build_object(
    'user_id', s.user_id,
    'profile_name', s.profile_name,
    'avatar_path', coalesce(s.avatar_path,''),
    'rank_name', public.bss_rank_name(s.rank_rp),
    'status', s.relationship
  ) order by s.sort_key, lower(s.profile_name),s.user_id),'[]'::jsonb)
  into result_players
  from (
    select p.user_id, p.profile_name, p.avatar_path, p.rank_rp,
      case
        when f.accepted_at is not null then 'friend'
        when f.requested_by = me then 'sent'
        when f.requested_by is not null then 'incoming'
        else 'none'
      end as relationship,
      case
        when lower(p.profile_name) = term then 0
        when starts_with(lower(p.profile_name),term) then 1
        else 2
      end as sort_key
    from public.bss_profiles p
    left join public.bss_friendships f
      on f.user_low = least(me,p.user_id) and f.user_high = greatest(me,p.user_id)
    where p.user_id <> me
      and position(term in lower(p.profile_name)) > 0
    order by sort_key,lower(p.profile_name),p.user_id
    limit page_size offset start_at
  ) s;

  return jsonb_build_object(
    'players',result_players,
    'total',total_count,
    'has_more',(start_at+jsonb_array_length(result_players))<total_count
  );
end;
$$;

revoke all on function public.bss_social_search_live(text,integer,integer) from PUBLIC, anon;
grant execute on function public.bss_social_search_live(text,integer,integer) to authenticated;

-- Cache des Supabase REST-API aktualisieren (PGRST202):
NOTIFY pgrst, 'reload schema';
