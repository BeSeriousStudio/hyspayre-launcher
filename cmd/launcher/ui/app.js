'use strict';
const tok = window.HYSPAYRE_LOCAL_TOKEN;
const allowedSite = 'https://hyspayre.com';
const $ = id => document.getElementById(id);
let snapshot = null;
let view = 'discover';
let notificationTimer;
function safe(s){return String(s??'').replace(/[&<>"']/g,x=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[x]));}
function toast(message){const el=$('toast');el.textContent=message;el.classList.add('show');clearTimeout(notificationTimer);notificationTimer=setTimeout(()=>el.classList.remove('show'),5500);}
async function api(path,method='GET',data){const res=await fetch(path,{method,headers:{'X-Hyspayre-Token':tok,...(data?{'Content-Type':'application/json'}:{})},body:data?JSON.stringify(data):undefined,cache:'no-store'});let out=await res.json();if(!res.ok)throw new Error(out.error||'Unbekannter Fehler');return out;}
function size(n){if(!n)return '—';const units=['B','KB','MB','GB'];let x=Number(n),i=0;while(x>=1024&&i<3){x/=1024;i++}return x.toFixed(i>1?1:0)+' '+units[i];}
function versionNewer(game){const inst=snapshot?.installed?.[game.id];return inst && inst.version!==game.version && game.status==='released';}
function statusText(game){const inst=snapshot?.installed?.[game.id];const t=snapshot?.tasks?.[game.id];if(t?.stage==='download')return 'WIRD GELADEN';if(t?.stage==='install')return 'INSTALLATION';if(inst && versionNewer(game))return 'UPDATE VERFÜGBAR';if(inst)return 'INSTALLIERT';if(game.status==='released')return 'VERFÜGBAR';return 'IN ENTWICKLUNG';}
function gamePicture(g){return g.image_url?g.image_url:'https://hyspayre.com/assets/card-architect.webp';}
function gameButton(g){const inst=snapshot?.installed?.[g.id];const task=snapshot?.tasks?.[g.id];if(task?.stage==='download'||task?.stage==='install')return `<button class="btn primary" disabled>${safe(task.message)}</button>`;
if(inst&&versionNewer(g))return `<button class="btn primary" data-action="install" data-game="${safe(g.id)}">UPDATE ${safe(g.version)} ⇩</button>`;
if(inst)return `<button class="btn primary" data-action="launch" data-game="${safe(g.id)}">JETZT SPIELEN ↗</button>`;
if(g.status==='released')return `<button class="btn primary" data-action="install" data-game="${safe(g.id)}">INSTALLIEREN · ${size(g.size)} ⇩</button>`;
return '<button class="btn line" disabled>DEMNÄCHST VERFÜGBAR</button>'}
function cardHtml(g){const tag=statusText(g);return `<article class="game-card"><div class="game-art"><img src="${safe(gamePicture(g))}" alt="Artwork zu ${safe(g.title)}"><span class="status-tag ${g.status==='upcoming'?'soon':''}">${safe(tag)}</span></div><div class="game-content"><div class="genre">${safe(g.genre||'HYSPAYRE GAME')}</div><h3>${safe(g.title)}</h3><p>${safe(g.description)}</p>${gameButton(g)}</div></article>`;}
function libraryHtml(g){const inst=snapshot.installed[g.id];const t=snapshot.tasks?.[g.id];return `<div class="game-row"><img src="${safe(gamePicture(g))}" alt=""><div class="detail"><h3>${safe(g.title)}</h3><p>${inst?'Installiert: '+safe(inst.version):'Noch nicht installiert'} · Aktuell: ${safe(g.version||'in Planung')}</p>${t?.stage==='error'?`<p style="color:#f5a9a9">${safe(t.message)}</p>`:''}${versionNewer(g)?`<p style="color:#8dcee8">Update verfügbar</p>`:''}</div><div class="row-actions">${gameButton(g)}${inst?`<button class="btn line" data-action="repair" data-game="${safe(g.id)}">REPARIEREN ↻</button><button class="btn line" data-action="uninstall" data-game="${safe(g.id)}">DEINSTALLIEREN</button>`:''}</div></div>`;}
function render(){if(!snapshot)return;
 $('side-version').textContent='CLIENT '+snapshot.version;
 const online=!snapshot.catalog_error;
 $('status-dot').classList.toggle('online',online);
 $('connection').textContent=online?'KATALOG VERBUNDEN':'OFFLINE-MODUS';
 $('bottom-status').textContent=online?'KATALOG AKTUELL':'OFFLINE · LOKALE BIBLIOTHEK';
 const games=snapshot.catalog.games||[],inst=snapshot.installed||{},tasks=snapshot.tasks||{};
 $('installed-count').textContent=Object.keys(inst).length;
 $('games-grid').innerHTML=games.map(cardHtml).join('')||'<div class="empty">Noch keine Spiele eingetragen.</div>';
 $('hero-status').textContent=games.some(g=>g.id==='hyspayre'&&g.status==='released')?'HYSPAYRE · JETZT VERFÜGBAR':'HYSPAYRE · IN ENTWICKLUNG';
 const hy=games.find(g=>g.id==='hyspayre');
 $('hero-primary').textContent=hy&&inst.hyspayre?'JETZT SPIELEN ↗':hy&&hy.status==='released'?'JETZT INSTALLIEREN ⇩':'DAS SPIEL ENTDECKEN ↗';
 $('library-grid').innerHTML=games.filter(g=>inst[g.id]||g.status==='released').map(libraryHtml).join('') || '<div class="empty"><h3>Deine Bibliothek wartet.</h3><p>Sobald ein Spiel veröffentlicht wird, kannst du es hier installieren.</p></div>';
 const all=games.filter(g=>g.status==='released'||tasks[g.id]);
 $('download-list').innerHTML=all.map(g=>{const t=tasks[g.id];const progress=t?.total?Math.min(100,t.done/t.total*100):0;return `<div class="download-task"><div class="download-task-head"><strong>${safe(g.title)}</strong><span>${safe(g.version||'')} · ${size(g.size)}</span></div><div><span>${t?safe(t.message):inst[g.id]?'Installiert · Version '+safe(inst[g.id].version):'Bereit zum Herunterladen'}</span>${t&&(t.stage==='download'||t.stage==='install')?`<div class="bar"><div style="width:${progress.toFixed(1)}%"></div></div><span>${progress.toFixed(0)}% · ${size(t.done)} / ${size(t.total)}</span>`:''}</div>${gameButton(g)}</div>`}).join('') || '<div class="empty">Aktuell stehen noch keine Spieldownloads zur Verfügung.</div>';
 $('install-root').value=document.activeElement?.id==='install-root'?$('install-root').value:snapshot.settings.install_root;
 $('client-ver').textContent='Version '+snapshot.version;
 const rel=snapshot.catalog.launcher;
 const hasUpdate=rel&&compareVersion(rel.version,snapshot.version)>0&&rel.url;
 $('update-client').classList.toggle('hidden',!hasUpdate);
 $('update-alert').classList.toggle('hidden',!hasUpdate);
 $('update-description').textContent=hasUpdate?'Version '+rel.version+' ist verfügbar. Dein Client kann sich selbst aktualisieren.':online?'Dein Client ist auf dem neuesten Stand.':'Der Online-Katalog ist nicht erreichbar. Du kannst es später erneut versuchen.';
}
function compareVersion(a,b){const p=String(a||'0').split('.').map(Number),q=String(b||'0').split('.').map(Number);for(let i=0;i<3;i++){if((p[i]||0)!==(q[i]||0))return (p[i]||0)>(q[i]||0)?1:-1}return 0;}
function showView(next){view=next;document.querySelectorAll('.view').forEach(e=>e.classList.toggle('active',e.id==='view-'+view));document.querySelectorAll('.navitem').forEach(e=>e.classList.toggle('selected',e.dataset.view===view));$('view-breadcrumb').textContent=({discover:'ENTDECKEN',library:'MEINE BIBLIOTHEK',downloads:'DOWNLOADS & UPDATES',community:'FREUNDE & SPIELER',account:'MEIN SPIELACCOUNT',settings:'EINSTELLUNGEN'})[view]||view;document.querySelector('.content').scrollTop=0;if(next==='account')socialOpenAccount();if(next==='community')socialOpenCommunity();}
async function refresh(){try{snapshot=await api('/api/bootstrap');render();}catch(e){toast('Client-Verbindung: '+e.message);}}
async function action(type,id){try{
 if(type==='uninstall'&&!confirm('Möchtest du '+id+' wirklich deinstallieren? Gespeicherte Spielstände in Godot werden nicht gelöscht.'))return;
 await api('/api/action','POST',{action:type,game_id:id});
 if(type==='install'||type==='repair'){showView('downloads');}
 await refresh();
 }catch(e){toast(e.message)} }
document.addEventListener('click',async e=>{
 const nav=e.target.closest('[data-view]');if(nav){showView(nav.dataset.view);return;}
 const external=e.target.closest('[data-open]');if(external){const url=external.dataset.open;if(url.startsWith(allowedSite+'/')){try{await api('/api/open','POST',{url});}catch(err){toast(err.message)}}return;}
 const btn=e.target.closest('[data-action]');if(btn){action(btn.dataset.action,btn.dataset.game);return;}
});
$('hero-primary').addEventListener('click',()=>{const hy=snapshot?.catalog.games.find(g=>g.id==='hyspayre');if(!hy)return;if(snapshot.installed.hyspayre)action('launch','hyspayre');else if(hy.status==='released')action('install','hyspayre');else showView('community');});
$('settings-form').addEventListener('submit',async e=>{e.preventDefault();try{await api('/api/settings','POST',{install_root:$('install-root').value});toast('Spielverzeichnis gespeichert.');await refresh();}catch(err){toast(err.message)}});
$('refresh-catalog').addEventListener('click',async()=>{toast('Katalog wird geprüft ...');try{await api('/api/refresh','POST',{});await refresh();toast('Aktualisierung abgeschlossen.');}catch(err){toast(err.message)}});
$('update-client').addEventListener('click',async()=>{if(!confirm('Neuen Client herunterladen und den Launcher neu starten?'))return;try{await api('/api/action','POST',{action:'launcher_update'});showView('downloads');toast('Client-Update wird geladen.');}catch(err){toast(err.message)}});
$('close-btn').addEventListener('click',async()=>{try{await api('/api/exit','POST',{});}catch{}window.close();});
refresh();setInterval(refresh,1700);

// HYSPAYRE SOCIAL · the same Supabase RPCs as hyspayre.com/spieler.html.
// All Supabase credentials stay in the native process; this view only receives
// public profile objects or the current user's own bss_me() response.
let selfProfile = null;
let socialReady = false;
let searched = '';
let results = [];
let resultTotal = 0;
let searchEpoch = 0;
let searchTimeout;
let focusedPlayerID = null;
let activeSocialAction = false;
const socialUUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
const SUPABASE_ASSETS = 'https://amgpdukqacxogafqpmho.supabase.co/storage/v1/object/public/bss-avatars/';
function avatarFor(p, cls='small-avatar'){
  const initials = safe(String(p.profile_name||'?').slice(0,1).toUpperCase());
  const path=String(p.avatar_path||'');
  if (!socialUUID.test(String(p.user_id||'')) || !new RegExp('^'+p.user_id+'/[0-9]{13}\\.png$','i').test(path)) return `<span class="${cls}">${initials}</span>`;
  const url=SUPABASE_ASSETS+path;
  return `<span class="${cls}"><img src="${safe(url)}" alt="Profilbild" onerror="this.remove()">${initials}</span>`;
}
function socialButtons(p){
  const id=safe(p.user_id), st=p.status;
  const btn=(type, label, primary=false)=>`<button type="button" class="btn ${primary?'primary':'line'} mini" data-friend-action="${type}" data-friend-id="${id}">${label}</button>`;
  if(st==='self')return '';
  if(st==='friend')return btn('remove','ENTFERNEN');
  if(st==='sent')return btn('cancel','ZURÜCKZIEHEN');
  if(st==='incoming')return btn('accept','ANNEHMEN',true)+btn('decline','ABLEHNEN');
  return btn('request','HINZUFÜGEN',true);
}
function socialPerson(p){
  return `<article class="person-row">${avatarFor(p)}<div class="person-main"><strong>${safe(p.profile_name||'Spieler')}</strong><small>${safe(p.rank_name||'Bronze 1')}${p.online===true?' · Online':''}</small></div><div class="person-buttons"><button class="icon-control" data-player="${safe(p.user_id)}" title="Profil ansehen" aria-label="Profil ansehen">↗</button>${socialButtons(p)}</div></article>`;
}
function renderSelf(profile){
  selfProfile=profile;
  const name=profile.profile_name||'Spieler';
  const rank=profile.rank||{};
  $('self-name').textContent=name;
  $('self-email').textContent=profile.email||'';
  $('self-rank').textContent=rank.name||'Bronze 1';
  $('self-rank-tag').textContent=(rank.name||'Bronze 1').toUpperCase();
  $('self-rp').textContent=`${Number(rank.progress)||0} / ${Number(rank.required)||100} RP · ${Number(rank.rp)||0} insgesamt`;
  $('self-rp-fill').style.width=`${Math.min(100,Math.max(0,(Number(rank.progress)||0)/Math.max(1,Number(rank.required)||100)*100))}%`;
  $('self-coins').textContent=(Number(profile.coins)||0).toLocaleString('de-DE');
  $('self-matches').textContent=Number(rank.matches)||0;
  $('self-wins').textContent=(Number(rank.wins)||0)+' Siege';
  $('self-avatar').innerHTML=avatarFor(profile,'large-avatar');
  $('top-account').innerHTML=safe(name)+' <span>♙</span>';
}
async function socialOpenAccount(){
  try{
    const status=await api('/api/auth/status');
    socialReady=!!status.authenticated;
    $('login-panel').classList.toggle('hidden',socialReady);
    $('account-panel').classList.toggle('hidden',!socialReady);
    $('community-locked').classList.toggle('hidden',socialReady);
    $('community-panel').classList.toggle('hidden',!socialReady);
    if(socialReady && status.profile)renderSelf(status.profile);
    else {$('top-account').innerHTML='MEIN ACCOUNT <span>♙</span>';selfProfile=null;}
  }catch(e){toast('Account: '+e.message);}
}
async function socialOpenCommunity(){
  await socialOpenAccount();
  if(!socialReady)return;
  await socialRefreshFriends();
}
async function socialRefreshFriends(){
  if(!socialReady)return;
  try{
    const response=await api('/api/social/friends');
    const all=Array.isArray(response.connections)?response.connections:[];
    const groups={friend:[],incoming:[],sent:[]};
    for(const person of all) if(groups[person.status])groups[person.status].push(person);
    const render=(id,list,empty)=>{$(id).innerHTML=list.length?list.map(socialPerson).join(''):`<p class="muted-msg">${empty}</p>`};
    render('friends-list',groups.friend,'Noch keine Freunde. Suche nach einem Spielernamen.');
    render('incoming-list',groups.incoming,'Keine offenen Anfragen.');
    render('sent-list',groups.sent,'Keine gesendeten Anfragen.');
    $('friends-total').textContent=groups.friend.length;
    $('incoming-total').textContent=groups.incoming.length;
    $('sent-total').textContent=groups.sent.length;
    $('requests-count').textContent=groups.incoming.length;
    $('requests-count').classList.toggle('hidden',groups.incoming.length===0);
  }catch(e){toast('Freundesliste: '+e.message);}
}
function socialScheduleSearch(){
  clearTimeout(searchTimeout);
  searched=$('people-query').value.trim();
  ++searchEpoch;results=[];resultTotal=0;
  $('people-more').classList.add('hidden');
  if(!searched){$('search-results').innerHTML='<p class="muted-msg">Gib einen Spielernamen ein.</p>';$('search-count').textContent='';return;}
  $('search-results').innerHTML='<p class="muted-msg">Spieler werden gesucht ...</p>';
  searchTimeout=setTimeout(()=>socialSearch(false),230);
}
async function socialSearch(more=false){
  if(!socialReady||!searched)return;
  const epoch=++searchEpoch, term=searched, offset=more?results.length:0;
  if(more)$('people-more').disabled=true;
  try{
    const page=await api('/api/social/search','POST',{query:term,offset,limit:40});
    if(epoch!==searchEpoch||term!==searched)return;
    const next=Array.isArray(page.players)?page.players:[];
    results=more?[...results,...next]:next;
    resultTotal=Math.max(results.length,Number(page.total)||0);
    $('search-count').textContent=page.legacy?`${results.length} Treffer (eingeschränkte Suche)`: `${results.length} von ${resultTotal} Treffern`;
    $('search-results').innerHTML=(page.notice?`<p class="muted-msg search-notice">${safe(page.notice)}</p>`:'')+(results.length?results.map(socialPerson).join(''):'<p class="muted-msg">Keine Spieler gefunden.</p>');
    $('people-more').classList.toggle('hidden',!(page.has_more||results.length<resultTotal));
  }catch(e){if(epoch===searchEpoch) $('search-results').innerHTML=`<p class="muted-msg">${safe(e.message)}</p>`;}
  finally{$('people-more').disabled=false;}
}
async function socialShowPlayer(id){
  if(!socialUUID.test(id))return;
  focusedPlayerID=id;
  const panel=$('social-profile');
  panel.classList.remove('hidden');panel.innerHTML='<p class="muted-msg">Profil wird geladen ...</p>';
  try{
    const p=await api('/api/social/profile','POST',{user_id:id});
    if(focusedPlayerID!==id)return;
    const rankName=p.rank_name||'Bronze 1';
    const required=Math.max(1,Number(p.rank_required)||100),progress=Math.max(0,Number(p.rank_progress)||0);
    panel.innerHTML=`<div class="social-detail-head"><div class="eyebrow muted">ÖFFENTLICHES SPIELERPROFIL</div><button id="close-profile" class="icon-control" aria-label="Profil schließen">✕</button></div>
      <div class="profile-summary">${avatarFor(p,'large-avatar')}<div><h2>${safe(p.profile_name||'Spieler')}</h2><p>${safe(rankName)} · ${p.online===true?'Online':p.status==='friend'?'Befreundet':'Hyspayre Spieler'}</p></div></div>
      <div class="player-stats"><div><span>RANG</span><strong>${safe(rankName)}</strong><small>${progress} / ${required} RP</small><div class="rank-track"><i style="width:${Math.min(100,progress/required*100)}%"></i></div></div><div><span>MATCHES</span><strong>${Number(p.matches)||0}</strong></div><div><span>SIEGE</span><strong>${Number(p.wins)||0}</strong></div></div>
      <div class="actions-inline">${socialButtons(p)}<button class="btn line" data-open="https://hyspayre.com/spieler.html?profil=${encodeURIComponent(id)}">PROFIL IM WEB ↗</button></div><p class="hint">E-Mail-Adresse, Guthaben und privates Inventar sind nicht öffentlich.</p>`;
  }catch(e){if(focusedPlayerID===id)panel.innerHTML=`<p class="muted-msg">${safe(e.message)}</p>`;}
}
async function socialFriendAction(id,actionType){
  if(activeSocialAction || !socialUUID.test(id))return;
  if(actionType==='remove'&&!confirm('Diese Freundschaft wirklich entfernen?'))return;
  activeSocialAction=true;
  try{
    await api('/api/social/action','POST',{user_id:id,action:actionType});
    toast(({request:'Anfrage gesendet',accept:'Freundschaft angenommen',decline:'Anfrage abgelehnt',cancel:'Anfrage zurückgezogen',remove:'Freund entfernt'})[actionType]||'Änderung gespeichert');
    await socialRefreshFriends();
    if(searched){++searchEpoch;await socialSearch(false);}
    if(focusedPlayerID)await socialShowPlayer(focusedPlayerID);
  }catch(e){toast(e.message)}
  finally{activeSocialAction=false;}
}
$('login-form').addEventListener('submit',async ev=>{
  ev.preventDefault();
  const button=$('login-submit');button.disabled=true;button.textContent='ANMELDUNG ...';
  const email=$('login-email').value,password=$('login-password').value,remember=$('login-remember').checked;
  try{
    await api('/api/auth/login','POST',{email,password,remember});
    $('login-password').value='';
    await socialOpenAccount();
    toast('Du bist angemeldet.');
  }catch(e){$('login-password').value='';toast('Anmeldung: '+e.message)}
  finally{button.disabled=false;button.textContent='ANMELDEN ↗';}
});
$('logout-btn').addEventListener('click',async()=>{
  try{await api('/api/auth/logout','POST',{});socialReady=false;selfProfile=null;searched='';$('people-query').value='';++searchEpoch;focusedPlayerID=null;$('social-profile').classList.add('hidden');await socialOpenAccount();toast('Abgemeldet.');}
  catch(e){toast('Abmelden: '+e.message)}
});
$('friends-refresh').addEventListener('click',socialRefreshFriends);
$('people-query').addEventListener('input',socialScheduleSearch);
$('people-more').addEventListener('click',()=>socialSearch(true));
document.addEventListener('click',e=>{
  const profile=e.target.closest('[data-player]');if(profile){socialShowPlayer(profile.dataset.player);return;}
  const act=e.target.closest('[data-friend-action]');if(act){socialFriendAction(act.dataset.friendId,act.dataset.friendAction);return;}
  if(e.target.closest('#close-profile')){$('social-profile').classList.add('hidden');focusedPlayerID=null;}
});
// Keep the account status in sync on first launch, not only when tab is opened.
socialOpenAccount();
