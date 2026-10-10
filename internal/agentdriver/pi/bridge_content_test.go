package pi

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	driverprotocol "github.com/regb/workitem/agentdriver/protocol"
	"github.com/regb/workitem/internal/testutil"
)

// Run the actual embedded TUI bridge with a fake Pi extension context. This
// needs Node's type stripping, not a live TUI, tmux, credentials, or a model.
func TestTUIBridgeContentAndBusyGuard(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node not installed")
	}
	if err := exec.Command(node, "--experimental-strip-types", "--eval", "").Run(); err != nil {
		t.Skip("Node type stripping unavailable")
	}
	root := testutil.ShortTempDir(t)
	bridge := filepath.Join(root, "bridge.ts")
	if err := os.WriteFile(bridge, bridgeSource, 0600); err != nil {
		t.Fatal(err)
	}
	script := fmt.Sprintf(`import bridge from %q;
import {createServer,createConnection} from 'node:net';
import assert from 'node:assert/strict';
const callbacks = new Map(); let idle = true; let shutDown = false; let leaf = 'branch-leaf';
const eventPath = process.env.WI_AGENT_EVENT_SOCKET;
const relay=createServer(socket=>{let data='';socket.on('data',chunk=>{data+=chunk;if(data.includes('\n')) socket.end('{"accepted":true}\n');});});
await new Promise(resolve=>relay.listen(eventPath,resolve));
const ctx={isIdle:()=>idle,hasPendingMessages:()=>false,shutdown:()=>{shutDown=true},sessionManager:{getLeafId:()=>leaf,getSessionId:()=> 'session',getSessionFile:()=> 'native'},ui:{notify:()=>{}}};
bridge({on:(name,fn)=>callbacks.set(name,fn),sendUserMessage:()=>{},events:{}});
await callbacks.get('session_start')({},ctx);
await new Promise(resolve=>setTimeout(resolve,60));
async function exchange(path,value){return await new Promise((resolve,reject)=>{const socket=createConnection(path);let data='';socket.on('error',reject);socket.on('connect',()=>socket.write(JSON.stringify(value)+'\n'));socket.on('data',chunk=>{data+=chunk;if(data.includes('\n')){socket.destroy();resolve(JSON.parse(data.split('\n')[0]));}});});}
const request={protocol_version:1,runtime_id:'tui-test',work_item_id:'item-test'};
callbacks.get('message_start')({message:{role:'assistant',content:[]}},ctx);
callbacks.get('message_update')({assistantMessageEvent:{type:'text_delta',delta:'hello 🌍',partial:{}}},ctx);
const path=process.env.WI_CONTENT_TEST_SOCKET;
let value=await exchange(path,request);assert.equal(value.text,'hello 🌍');assert.equal(value.leaf_id,'branch-leaf');
value=await exchange(path,request);assert.equal(value.text,'hello 🌍');
value=await exchange(path,{...request,runtime_id:'superseded'});assert.equal(value.error,'content owner mismatch');
value=await exchange(process.env.WI_AGENT_CONTROL_SOCKET,{...request,id:'send-first',type:'prompt',message:'work',require_idle:true});assert.equal(value.accepted,true);
value=await exchange(process.env.WI_AGENT_CONTROL_SOCKET,{...request,id:'shutdown-before-start',type:'shutdown',require_idle:true});assert.equal(value.accepted,false);assert.equal(shutDown,false);
callbacks.get('agent_start')({},ctx);callbacks.get('agent_settled')({},ctx);
idle=false;
value=await exchange(process.env.WI_AGENT_CONTROL_SOCKET,{...request,id:'shutdown-test',type:'shutdown',require_idle:true});assert.equal(value.accepted,false);assert.equal(shutDown,false);
callbacks.get('message_end')({message:{role:'assistant',content:[{type:'text',text:'saved answer'}],stopReason:'stop'}},ctx);
value=await exchange(path,request);assert.equal(value.text,'');
leaf='other-leaf';callbacks.get('session_tree')({oldLeafId:'branch-leaf',newLeafId:leaf},ctx);value=await exchange(path,request);assert.equal(value.leaf_id,leaf);
callbacks.get('session_shutdown')({},ctx);
await new Promise(resolve=>setTimeout(resolve,60));relay.close();process.exit(0);
`, (&url.URL{Scheme: "file", Path: bridge}).String())
	scriptPath := filepath.Join(root, "test.mjs")
	if err := os.WriteFile(scriptPath, []byte(script), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, node, "--experimental-strip-types", scriptPath)
	cmd.Env = append(os.Environ(), "WI_ID=item-test", "WI_AGENT_RUNTIME_ID=tui-test", "WI_AGENT_MODE=tui", "WI_AGENT_CONTROL_SOCKET="+filepath.Join(root, "control.sock"), "WI_AGENT_EVENT_SOCKET="+filepath.Join(root, "events.sock"), "WI_CONTENT_TEST_SOCKET="+driverprotocol.ContentSocketPath(filepath.Join(root, "control.sock")))
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("TUI bridge %v\n%s", err, output)
	}
}
