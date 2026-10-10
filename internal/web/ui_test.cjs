const assert = require('node:assert/strict');
const { test } = require('node:test');
const fs = require('node:fs');
const vm = require('node:vm');

async function ui() {
  const elements = new Map(), timers = new Map();
  let timerID = 0;
  function element() {
    return { value: '', disabled: false, hidden: true, children: [], textContent: '',
      replaceChildren(...children) { this.children = children; },
      append(...children) { this.children.push(...children); },
      setAttribute(name, value) { this[name] = value; },
      focus() {}, reportValidity() { return true; }, requestSubmit() {},
    };
  }
  const get = id => { if (!elements.has(id)) elements.set(id, element()); return elements.get(id); };
  const context = vm.createContext({
    document: { getElementById: get, createElement: element },
    location: { hash: '', search: '', pathname: '/' },
    history: { replaceState() {}, pushState() {} }, sessionStorage: { getItem() { return ''; } },
    window: { addEventListener() {} }, URLSearchParams,
    fetch: async () => { throw new Error('No network in UI tests'); },
    setTimeout(callback, delay) { const id = ++timerID; timers.set(id, { callback, delay }); return id; },
    clearTimeout(id) { timers.delete(id); },
  });
  vm.runInContext(fs.readFileSync(`${__dirname}/static/app.js`, 'utf8'), context);
  // Settle initial list/receipt requests before exercising the UI.
  for (let i = 0; i < 8; i++) await Promise.resolve();
  return { get, context, timers };
}

for (const [inputID, formID, buttonID] of [['message', 'composer', 'send'], ['new-prompt', 'new-form', 'create']]) {
  test(`${inputID}: Enter sends; Shift+Enter and IME do not`, async () => {
    const { get } = await ui();
    get('new-project').value = 'repo';
    let submissions = 0;
    get(formID).requestSubmit = button => { assert.equal(button, get(buttonID)); submissions++; };
    for (const scenario of [
      { key: 'Enter', sends: true, prevents: true },
      { key: 'Enter', shiftKey: true },
      { key: 'Enter', isComposing: true },
      { key: 'Enter', keyCode: 229 },
      { key: 'Enter', repeat: true, prevents: true },
      { key: 'Enter', disabled: true, prevents: true },
      { key: 'Enter', blank: true, prevents: true },
      { key: 'Escape' },
    ]) {
      get(inputID).value = scenario.blank ? '  ' : 'Task';
      get(buttonID).disabled = !!scenario.disabled;
      const before = submissions;
      let prevented = false;
      get(inputID).onkeydown({ ...scenario, preventDefault() { prevented = true; } });
      assert.equal(submissions - before, scenario.sends ? 1 : 0);
      assert.equal(prevented, !!scenario.prevents);
    }
  });
}

test('pending startup displays only a spinner without history or live reads', async () => {
  const { get, context, timers } = await ui();
  const requested=[];
  context.fetch=async path=>{requested.push(path);return {ok:true,json:async()=>({item:{id:'01K00000000000000000000000'},startup:{state:'starting'}})}};
  await vm.runInContext('openChat("01K00000000000000000000000")',context);
  assert.equal(get('chat-loading').hidden,false);
  assert.equal(get('chat').hidden,true);
  assert.equal(get('archive').hidden,true);
  assert.equal(get('notice').textContent,'');
  assert.equal(requested.length,1);
  assert.ok(requested[0].endsWith('/chat'));
  assert.ok([...timers.values()].some(timer=>timer.delay===500));
  vm.runInContext('stopChat()',context);
  assert.equal(get('chat-loading').hidden,true);
});

test('Enter without a repository marks the picker invalid without sending', async () => {
  const { get } = await ui();
  get('new-prompt').value = 'Task'; get('create').disabled = true;
  let submitted = false, focused = false;
  get('new-form').requestSubmit = () => { submitted = true; };
  get('new-project').focus = () => { focused = true; };
  get('new-prompt').onkeydown({ key: 'Enter', preventDefault() {} });
  assert.equal(submitted, false);
  assert.equal(focused, true);
  assert.equal(get('new-project')['aria-invalid'], 'true');
  assert.equal(get('new-project-error').hidden, false);
  get('new-project').value = 'repo'; get('new-project').onchange();
  assert.equal(get('new-project')['aria-invalid'], 'false');
  assert.equal(get('new-project-error').hidden, true);
});

test('creation receipt shows a six-second toast linked by immutable ID, not title', async () => {
  const { get, context, timers } = await ui();
  vm.runInContext(`newProjects=[{id:'repo'}];`, context);
  get('new-project').value = 'repo'; get('new-prompt').value = 'Task';
  const id = '01K00000000000000000000000';
  context.fetch = async () => ({ ok: true, json: async () => ({ id, title: 'New conversation', state: 'queued' }) });
  await get('new-form').onsubmit({ preventDefault() {} });
  const toast = get('started-toast');
  assert.equal(toast.hidden, false);
  assert.equal(toast.children[0].textContent, 'Work item started');
  const link = toast.children[1];
  assert.equal(link.href, `/?item=${id}`);
  assert.equal(get('new-prompt').value, '');
  assert.equal(context.location.pathname, '/');
  vm.runInContext('openChat=async id=>{window.openedID=id;};', context);
  let prevented = false;
  link.onclick({ button: 0, preventDefault() { prevented = true; } });
  assert.equal(prevented, true);
  assert.equal(context.window.openedID, id);
  const expiry = [...timers.values()].find(timer => timer.delay === 6000);
  assert.ok(expiry); expiry.callback();
  assert.equal(toast.hidden, true);
  assert.equal(toast.children.length, 0);
});
