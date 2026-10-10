'use strict';
const el = id => document.getElementById(id);
const hash = new URLSearchParams(location.hash.slice(1));
if (hash.get('token')) { sessionStorage.setItem('wi-token', hash.get('token')); history.replaceState(null, '', '/'); }
const token = sessionStorage.getItem('wi-token') || '';
let lastChat;
let item = '', runtime = '', generation = 0, streaming, refreshTimer, pending = false, lastHistory = '';
const drafts = new Map();
let creating = false, creationUnknown = false, newProjects = [], lastLaunchErrors = '';
const pendingLaunches = new Map();
let startedToastTimer;
function showStartedToast(id) {
  const toast=el('started-toast'), link=node('a','Open conversation');
  link.href=`/?item=${encodeURIComponent(id)}`;
  link.onclick=event=>{
    if(event.button!==0 || event.ctrlKey || event.metaKey || event.shiftKey || event.altKey)return;
    event.preventDefault();openChat(id).catch(error=>notice(error.message));
  };
  clearTimeout(startedToastTimer);
  toast.replaceChildren(node('span','Work item started'),link);toast.hidden=false;
  startedToastTimer=setTimeout(()=>{toast.hidden=true;toast.replaceChildren();},6000);
}
let listData, selectedProjects = null; // null tracks all projects, including newly added ones.
function node(tag, text, className) {
  const element = document.createElement(tag);
  if (text !== undefined) element.textContent = text;
  if (className) element.className = className;
  return element;
}
function statusBadge(status) { return node('span', status || 'unknown', `badge status-${status || 'unknown'}`); }
function labels(values, capacityFull=false) {
  const container = node('div');
  if (capacityFull) container.append(node('span', 'capacity full', 'label capacity-full'));
  for (const value of values || []) container.append(node('span', value, 'label'));
  return container;
}
function projectQuery() {
  const query = new URLSearchParams();
  if (selectedProjects !== null) {
    for (const id of selectedProjects) query.append('project', id);
    if (!selectedProjects.size) query.append('project', ''); // Explicitly empty, not all.
  }
  return query;
}
function updateProjectSummary() {
  const projects = listData?.projects || [];
  const count = projects.filter(p => selectedProjects === null || selectedProjects.has(p.id)).length;
  el('project-summary').textContent = selectedProjects === null ? 'Projects · All projects' : `Projects · ${count} selected`;
  for (const input of el('project-options').querySelectorAll('input')) input.checked = selectedProjects === null || selectedProjects.has(input.value);
}
function renderProjects() {
  el('project-options').replaceChildren();
  for (const project of listData.projects || []) {
    const label = node('label', undefined, 'project-option');
    const input = node('input'); input.type = 'checkbox'; input.value = project.id;
    const name = node('span', project.name);
    input.onchange = () => {
      if (selectedProjects === null) selectedProjects = new Set(listData.projects.map(p => p.id));
      if (input.checked) selectedProjects.add(project.id); else selectedProjects.delete(project.id);
      updateProjectSummary(); renderList();
    };
    label.append(input, name); el('project-options').append(label);
  }
  if (!el('project-options').childElementCount) el('project-options').append(node('p', 'No projects.'));
  updateProjectSummary();
}
function renderList() {
  const pane = el('work-list'); pane.replaceChildren();
  if (!listData) return;
  el('capacity').textContent = `Deep work active: ${listData.deep_work.active}/${listData.deep_work.limit}`;
  const table = node('table'); table.setAttribute('aria-label', 'Work items');
  const head = node('thead'), headings = node('tr');
  for (const title of ['Item', 'Agent', 'Worktree', 'Deep', 'Labels', 'Repository', 'Actions']) headings.append(node('th', title));
  head.append(headings); table.append(head);
  let count = 0;
  for (const section of listData.display_sections || []) {
    const rows = section.items.filter(row => selectedProjects === null || selectedProjects.has(row.project_id));
    if (!rows.length) continue;
    const body = node('tbody'), group = node('tr', undefined, `bucket ${section.name.toLowerCase().replaceAll(' ', '-')}`);
    const heading = node('th', section.name); heading.colSpan = 7; heading.scope = 'rowgroup'; heading.append(node('small', String(rows.length)));
    group.append(heading); body.append(group);
    for (const row of rows) {
      const tr = node('tr', undefined, 'row'); tr.dataset.item = row.id;
      const open = button('', () => openChat(row.id), 'open'); open.title = row.title;
      open.append(node('span', row.title, 'item-name'));
      const name = node('td'); name.append(open);
      const agent = node('td'); agent.append(statusBadge(row.agent?.status)); agent.title = row.agent?.reason || '';
      const worktree = node('td'); worktree.append(statusBadge(row.worktree?.status || 'absent')); worktree.title = row.worktree?.reason || '';
      const deep = node('td', row.deep_work ? '●' : '', 'deep'); deep.setAttribute('aria-label', row.deep_work ? 'Deep work' : 'Not deep work');
      const tags = node('td'); tags.append(labels(row.labels, row.capacity_full));
      const projectName = listData.projects.find(project => project.id === row.project_id)?.name || 'Unknown project';
      const repo = node('td', `${projectName}${row.repository_home ? ' (home)' : ''}`, 'repository');
      const actions = node('td'), archiveButton = button('Archive', () => archive(row.id, row.title), 'archive-row');
      archiveButton.setAttribute('aria-label', `Archive ${row.title}`); actions.append(archiveButton);
      tr.append(name, agent, worktree, deep, tags, repo, actions); body.append(tr); count++;
    }
    table.append(body);
  }
  if (count) pane.append(table);
  else pane.append(node('p', selectedProjects === null ? 'No active work items.' : 'No work items in the selected projects. Select all to see every project.', 'meta'));
}
function renderTicket(data) {
  const m = data.item, live = data.runtime.runtime;
  el('ticket-title').textContent = m.title;
  el('ticket-description').textContent = data.description || 'No description.';
  el('ticket-status').replaceChildren(node('span', m.state, 'badge'), statusBadge(data.status));
  el('ticket-labels').replaceChildren(labels(m.labels));
  if (!m.labels?.length) el('ticket-labels').append(node('span', 'No labels', 'meta'));
  const info = el('ticket-info'); info.replaceChildren();
  const field = (name, value, code=false) => {
    info.append(node('dt', name), node('dd', value || 'None', code ? 'code' : ''));
  };
  field('Project', data.project.name);
  if (m.repository.remote_url) field('Origin', m.repository.remote_url, true);
  field('Item', m.id, true);
  if (m.slug) field('Slug', m.slug, true);
  field('Branch', m.checkout.specification.branch, true);
  field('Checkout', m.checkout.specification.kind);
  field('Workspace', m.checkout.claim?.path || 'Not materialized', true);
  field('Deep work', m.deep_work ? 'Yes' : 'No');
  field('Agent', m.agent);
  field('Runtime', live ? `${live.mode} · ${live.id}` : 'No runtime', true);
  if (m.primary_conversation) field('Conversation', m.primary_conversation.id, true);
  field('Created', new Date(m.created_at).toLocaleString());
  field('Updated', new Date(m.updated_at).toLocaleString());
  field('Created from', m.repository.created_from_commit, true);
}
function validateNewRepository() {
  const missing=!el('new-project').value;
  el('new-project').setAttribute('aria-invalid',String(missing));
  el('new-project-error').hidden=!missing;
  if(missing)el('new-project').focus();
  return !missing;
}
function updateNewForm() {
  if(el('new-project').value)validateNewRepository();
  for (const id of ['new-project', 'new-prompt']) el(id).disabled = creating || creationUnknown;
  el('create').disabled = creating || creationUnknown || !newProjects.length || !el('new-project').value || !el('new-prompt').value.trim();
  el('create').textContent = creating ? '…' : '↑';
}
function showUnknownCreation() {
  const result = el('creation-result');
  result.replaceChildren(node('p', 'The creation result is unknown. A ticket may already exist. Check the work list before trying again.'));
  result.append(button('I checked the list; allow retry', async () => {
    if (!confirm('A ticket may already exist. Have you checked the work list? Retrying could create a duplicate.')) return;
    creationUnknown = false; result.replaceChildren(); updateNewForm();
  }));
}
async function newPage(navigate=true) {
  const contextProject = lastChat?.project.id, previous = el('new-project').value;
  stopChat(); item=''; runtime=''; setLocation('/?new=1',navigate);
  el('items').hidden=true; el('chat').hidden=true; el('new-page').hidden=false;
  el('archive').hidden=true; el('title').textContent='New conversation'; notice('');
  el('create').disabled=true; el('new-project').disabled=true;
  const gen=generation;
  try {
    const projects=await api('/api/projects'); if(gen!==generation)return;
    newProjects=projects;
    el('new-project').replaceChildren(node('option', 'Choose a repo'));
    el('new-project').firstChild.value='';
    for (const project of projects) { const option=node('option', project.name); option.value=project.id; el('new-project').append(option); }
    const scoped=projects.filter(p=>selectedProjects?.has(p.id));
    const hasDraft=el('new-prompt').value;
    const candidates=hasDraft && previous ? [previous] : [contextProject, scoped.length===1 ? scoped[0].id : '', previous, projects.length===1 ? projects[0].id : ''];
    el('new-project').value=candidates.find(id=>projects.some(p=>p.id===id)) || '';
    updateNewForm();
    if (!projects.length) notice('No registered repositories yet. Register one with wi repo add NAME --from SOURCE, then reopen New.');
    if (creationUnknown) showUnknownCreation();
    else el('new-prompt').focus();
  } catch(e) { if(gen===generation)notice(e.message); }
}
function notice(text) { el('notice').textContent = text || ''; }
async function api(path, body, signal) {
  const response = await fetch(path, {method: body === undefined ? 'GET' : 'POST', headers: {'Authorization': `Bearer ${token}`, ...(body === undefined ? {} : {'Content-Type':'application/json'})}, body: body === undefined ? undefined : JSON.stringify(body), signal});
  const value = await response.json();
  if (!response.ok) { const err = new Error([value.error, ...(value.warnings || [])].filter(Boolean).join('\n')); err.outcome = value; throw err; }
  return value;
}
function button(text, action, className) { const b = document.createElement('button'); b.textContent = text; b.className = className || ''; b.onclick = () => action().catch(e => notice(e.message)); return b; }
function stopChat() { lastChat=undefined; generation++; streaming?.abort(); streaming = undefined; el('chat-loading').hidden=true; clearTimeout(refreshTimer); if (item) drafts.set(item, el('message').value); }
function setLocation(url,navigate) { if(navigate && location.pathname+location.search!==url) history.pushState(null,'',url); }
async function list(navigate=true) {
  stopChat(); item = ''; runtime = ''; setLocation('/',navigate); el('chat').hidden = true; el('new-page').hidden = true; el('items').hidden = false; el('archive').hidden = true; el('title').textContent = 'Work items';
  const gen = generation; const data = await api('/api/items'); if(gen !== generation) return;
  listData = data; notice((data.warnings || []).join('\n'));
  renderProjects(); renderList();
}
async function openChat(id,navigate=true) {
  clearTimeout(startedToastTimer);el('started-toast').hidden=true;
  stopChat(); item=id;runtime='';lastHistory='';el('message').value=drafts.get(id)||'';setLocation(`/?item=${encodeURIComponent(id)}`,navigate);
  el('items').hidden=true;el('new-page').hidden=true;el('chat').hidden=true;el('chat-loading').hidden=false;el('archive').hidden=true;el('title').textContent='Loading conversation';el('ticket-title').textContent='Loading ticket';el('ticket-description').textContent='';el('ticket-status').replaceChildren();el('ticket-info').replaceChildren();el('ticket-labels').replaceChildren();el('history').replaceChildren();el('live').textContent='';notice('');el('send').disabled=true;
  const gen=generation;await refresh(gen);
}
async function refresh(gen) {
  if(gen!==generation)return;
  try {
    const data=await api(`/api/items/${item}/chat`);if(gen!==generation)return;
    if(['queued','starting'].includes(data.startup?.state)) {
      el('chat-loading').hidden=false;el('chat').hidden=true;el('archive').hidden=true;notice('');
      return;
    }
    el('chat-loading').hidden=true;el('chat').hidden=false;el('archive').hidden=false;
    el('title').textContent=data.item.title;
    if(data.startup?.state==='problem')notice(data.startup.error);
    renderTicket(data);
    runtime=data.runtime.runtime?.id || '';
    el('owner').textContent=`${data.status} · ${data.runtime.online ? `${data.runtime.runtime.mode} · verified primary-agent owner` : (data.item.state==='archived' ? 'Read-only archived conversation.' : pendingLaunches.has(data.item.id) ? 'Starting the primary agent in the background…' : 'No verified live owner. Start one explicitly with the CLI.')}\n${(data.warnings||[]).join('\n')}`;
    el('archive').disabled=data.item.state==='archived';
    lastChat=data;
    updateComposer();
    el('message').disabled=data.item.state==='archived';
    if(data.status==='busy' || !data.idle_guard) el('delivery').value='follow_up';
    updateComposer();
    const key=JSON.stringify(data.history);if(key!==lastHistory){
      const pane=el('history');const bottom=pane.scrollHeight-pane.scrollTop-pane.clientHeight<80;const pos=pane.scrollTop;pane.replaceChildren();
      for(const m of data.history.messages||[]) {const div=document.createElement('article');div.className=`message ${m.role}`;const role=document.createElement('div');role.className='role';role.textContent=m.role;const text=document.createElement('div');text.className='text';text.textContent=m.text;div.append(role,text);pane.append(div);}
      if(data.history.truncated){const note=document.createElement('p');note.textContent='Showing the latest 200 messages, with bounded text. Older or oversized history is omitted in this prototype.';pane.prepend(note);}
      if(!pane.childElementCount)pane.textContent='No recorded messages.';pane.scrollTop=bottom||!lastHistory?pane.scrollHeight:pos;lastHistory=key;
    }
    if(!streaming)stream(gen);
  } catch(e){if(gen===generation){el('chat-loading').hidden=true;notice(e.message);}}
  finally {if(gen===generation){clearTimeout(refreshTimer);refreshTimer=setTimeout(()=>refresh(gen),el('chat-loading').hidden?3000:500);}}
}
async function stream(gen) {
  const controller=new AbortController();streaming=controller;
  try {
    const response=await fetch(`/api/items/${item}/stream`,{headers:{Authorization:`Bearer ${token}`},signal:controller.signal});if(!response.ok)throw new Error('Live connection refused');
    const reader=response.body.getReader(), decoder=new TextDecoder();let buffer='';
    while(true){const {done,value}=await reader.read();if(done)break;buffer+=decoder.decode(value,{stream:true});let n;
      while((n=buffer.indexOf('\n'))>=0){const data=JSON.parse(buffer.slice(0,n));buffer=buffer.slice(n+1);if(gen!==generation)return;
        el('live').textContent=data.error || [data.activity,data.text].filter(Boolean).join('\n');
      }
    }
  } catch(e){if(e.name!=='AbortError'&&gen===generation)el('live').textContent=e.message;}
  if(gen===generation)setTimeout(()=>{if(gen===generation)stream(gen)},3000);
}
async function archive(id,title) {
  if(!confirm(`Archive ${title}? Wi will check workspace safety and refuse a busy agent. It may shut down an idle agent and release the checkout.`))return;
  notice('Checking archive safety…');
  try{const out=await api(`/api/items/${id}/archive`,{});await list();notice(['Archived.',...(out.warnings||[])].join('\n'));}
  catch(e){notice(e.message);if(id===item)await refresh(generation);}
}
function updateComposer() {
 const data=lastChat, mode=el('delivery').value;
 const supported=mode==='prompt' ? data?.runtime.capabilities.prompt && data?.idle_guard : data?.runtime.capabilities.follow_up;
 el('send').disabled=pending || !data?.runtime.online || data?.item.state==='archived' || !supported;
}
el('delivery').onchange=updateComposer;
function submitOnEnter(inputID,formID,submitID,validate=()=>true) {
 el(inputID).onkeydown=event=>{
  if(event.key!=='Enter' || event.shiftKey || event.isComposing || event.keyCode===229)return;
  event.preventDefault();
  if(event.repeat || el(inputID).disabled || !validate() || el(submitID).disabled || !el(inputID).value.trim())return;
  el(formID).requestSubmit(el(submitID));
 };
}
submitOnEnter('message','composer','send');
submitOnEnter('new-prompt','new-form','create',validateNewRepository);
el('new-project').oninvalid=validateNewRepository;
el('composer').onsubmit=async event=>{
 event.preventDefault();if(pending||!item)return;const id=item, gen=generation, text=el('message').value;pending=true;el('send').disabled=true;
 try{const out=await api(`/api/items/${id}/messages`,{text,delivery:el('delivery').value,runtime_id:runtime});drafts.delete(id);if(gen===generation){el('message').value='';notice('Submitted to the existing agent. Acceptance is not completion.');}}
 catch(e){if(gen===generation)notice(`${e.message}\nIf the connection failed, delivery may be unknown. Check history before retrying.`);}
 finally{pending=false;if(gen===generation)await refresh(gen);}
};
el('new-prompt').oninput=updateNewForm;
el('new-project').onchange=updateNewForm;
el('attention').onclick=()=>list().catch(e=>notice(e.message));
el('new').onclick=()=>newPage().catch(e=>notice(e.message));
el('new-form').onsubmit=async event=>{
  event.preventDefault(); if(creating || creationUnknown || !validateNewRepository() || !el('new-form').reportValidity())return;
  const request={project_id:el('new-project').value,prompt:el('new-prompt').value.trim()};
  if(!request.project_id || !request.prompt)return;
  const gen=generation;creating=true;updateNewForm();notice('Sending…');
  let outcome, error;
  try { outcome=await api('/api/items',request); }
  catch(e) { error=e;outcome=e.outcome; }
  creating=false;updateNewForm();
  if(outcome?.id){
    pendingLaunches.set(outcome.id, Date.now());
    showStartedToast(outcome.id);
    if(el('new-prompt').value.trim()===request.prompt)el('new-prompt').value='';
    el('creation-result').replaceChildren();updateNewForm();
    if(gen===generation){notice('');el('new-prompt').focus();}
  } else if(error){
    if(outcome?.observed?.id){
      if(el('new-prompt').value.trim()===request.prompt)el('new-prompt').value='';
      updateNewForm();
      if(gen===generation){
        el('creation-result').replaceChildren(node('p','Saved, but startup needs attention.'),button('View conversation',()=>openChat(outcome.observed.id)));
        el('new-prompt').focus();
      }
    }
    if(!outcome || outcome.creation_uncertain){creationUnknown=true;updateNewForm();if(gen===generation)showUnknownCreation();}
    if(gen===generation)notice(error.message);
  } else {creationUnknown=true;updateNewForm();if(gen===generation)showUnknownCreation();}
};
el('list').onclick=()=>list().catch(e=>notice(e.message));
el('projects-all').onclick=()=>{selectedProjects=null;updateProjectSummary();renderList();};
el('projects-none').onclick=()=>{selectedProjects=new Set();updateProjectSummary();renderList();};
el('next').onclick=async()=>{try{const gen=generation, query=projectQuery();query.set('current',item);const next=await api(`/api/next?${query}`);if(gen!==generation)return;if(next.id)await openChat(next.id);else notice(selectedProjects===null ? 'No item needs attention.' : 'No item needs attention in the selected projects.');}catch(e){notice(e.message);}};
el('archive').onclick=()=>archive(item,el('title').textContent);
// Background receipts and attention are cues, never automatic navigation.
async function pollBackground() {
  try {
    const [jobsResult, listResult]=await Promise.allSettled([api('/api/creations'),api('/api/items')]);
    const jobs=jobsResult.status==='fulfilled' ? jobsResult.value : [];
    for(const job of jobs)if(!['queued','starting'].includes(job.state))pendingLaunches.delete(job.id);
    for(const [id,started] of pendingLaunches)if(Date.now()-started>150000)pendingLaunches.delete(id);
    const observedList=listResult.status==='fulfilled' ? listResult.value : listData;
    const visibleRows=new Map();
    for(const section of observedList?.display_sections || [])for(const row of section.items)if(selectedProjects===null || selectedProjects.has(row.project_id))visibleRows.set(row.id,row);
    const problems=jobs.filter(job=>job.state==='problem' && (!observedList || visibleRows.has(job.id)));
    const key=JSON.stringify(problems);
    if(jobsResult.status==='fulfilled' && key!==lastLaunchErrors){
      lastLaunchErrors=key;el('launch-updates').replaceChildren();
      for(const job of problems.slice(0,3)){
        const row=node('div',undefined,'launch-problem');
        row.append(node('p',`${job.title}: ${job.error}`),button('View conversation',()=>openChat(job.id)));
        el('launch-updates').append(row);
      }
    }
    if(listResult.status==='fulfilled'){
      const data=listResult.value, pendingIDs=new Set([...pendingLaunches.keys(),...jobs.filter(job=>['queued','starting'].includes(job.state)).map(job=>job.id)]);
      const ids=new Set(problems.map(job=>job.id));
      for(const section of data.display_sections || []){
        if(!['NEEDS ATTENTION','NEEDS FIXING'].includes(section.name))continue;
        for(const row of section.items)if(!pendingIDs.has(row.id) && (selectedProjects===null || selectedProjects.has(row.project_id)))ids.add(row.id);
      }
      el('attention').hidden=!ids.size;el('attention').textContent=`Attention · ${ids.size}`;
      if(!el('items').hidden){listData=data;renderList();}
    }
  } catch(_) { /* Keep the last observation; do not interrupt the composer. */ }
  finally {setTimeout(pollBackground,5000);}
}
pollBackground();
function route() {
  const query=new URLSearchParams(location.search), id=query.get('item');
  return id ? openChat(id,false) : query.has('new') ? newPage(false) : list(false);
}
window.addEventListener('popstate',()=>route().catch(e=>notice(e.message)));
route().catch(e=>notice(e.message));
