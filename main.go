package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Note struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Subject   string    `json:"subject"`
	Content   string    `json:"content"`
	Summary   string    `json:"summary"`
	CreatedAt time.Time `json:"created_at"`
	HasImage  bool      `json:"has_image"`
	ImageType string    `json:"image_type,omitempty"`
}

const schema = `CREATE TABLE IF NOT EXISTS notes (
 id BIGSERIAL PRIMARY KEY,
 title TEXT NOT NULL CHECK(length(title)>0),
 subject TEXT NOT NULL DEFAULT '',
 content TEXT NOT NULL CHECK(length(content)>0),
 summary TEXT NOT NULL DEFAULT '',
 image_data BYTEA,
 image_type TEXT NOT NULL DEFAULT '',
 created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);`

type App struct{ DB *pgxpool.Pool }

func (a *App) routes() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		io.WriteString(w, page)
	})
	m.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := a.DB.Ping(ctx); err != nil {
			http.Error(w, "database unavailable", 503)
			return
		}
		respond(w, 200, map[string]string{"status": "ok"})
	})
	m.HandleFunc("POST /notes", a.create)
	m.HandleFunc("POST /notes/photo", a.createPhoto)
	m.HandleFunc("DELETE /notes/{id}", a.delete)
	m.HandleFunc("POST /notes/{id}/summarize", a.summary)
	m.HandleFunc("GET /notes/{id}/image", a.image)
	m.HandleFunc("GET /notes", a.list)
	m.HandleFunc("GET /export", a.export)
	return m
}

const page = `<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>MemoryBox · Study notes</title>
<style>
@import url('https://fonts.googleapis.com/css2?family=DM+Sans:wght@400;500;600;700&family=Manrope:wght@400;500;600;700;800&display=swap');
:root{color-scheme:light;--ink:#181818;--muted:#777;--paper:#f3f3f3;--card:#fff;--line:#e5e5e5;--black:#151515;--black-soft:#292929;--mist:#f5f5f5;--shadow:0 14px 40px #0000000b}*{box-sizing:border-box}body{margin:0;background:radial-gradient(ellipse at 85% 0%,#e8e8e8 0,transparent 30%),var(--paper);color:var(--ink);font:15px/1.55 'Manrope','DM Sans',system-ui,sans-serif}header{height:76px;border-bottom:1px solid #e7e7e7;background:#ffffffed;display:flex;align-items:center;justify-content:space-between;padding:0 max(calc((100vw - 1180px)/2),28px);position:sticky;top:0;z-index:5}.brand{font-weight:800;font-size:18px;display:flex;align-items:center;gap:11px;letter-spacing:-.04em}.logo{display:grid;place-items:center;width:38px;height:38px;border-radius:12px;background:var(--black);color:#fff;box-shadow:0 6px 15px #0002;font-size:19px}.status{font-size:11px;color:#424242;background:#f4f4f4;border:1px solid #e6e6e6;border-radius:30px;padding:8px 13px;font-weight:700}.wrap{max-width:1180px;margin:38px auto;padding:0 28px}.hero{position:relative;overflow:hidden;display:flex;justify-content:space-between;align-items:center;min-height:205px;margin-bottom:24px;padding:34px 40px;border-radius:22px;background:linear-gradient(112deg,#101010 0%,#242424 60%,#393939 100%);color:#fff;box-shadow:0 18px 36px #0002}.hero:after{content:'✳';position:absolute;right:19%;top:-100px;font-size:300px;line-height:1;color:#ffffff0a;pointer-events:none}.hero>div{position:relative;z-index:1}.eyebrow{text-transform:uppercase;letter-spacing:.18em;font-size:10px;color:#c9c9c9;font-weight:800}.hero h1{font-size:clamp(32px,4.3vw,48px);line-height:1.08;letter-spacing:-.06em;margin:9px 0 7px}.hero p{color:#d0d0d0;margin:0;font-size:14px}.count{z-index:1;background:#ffffff14;border:1px solid #ffffff30;padding:15px 20px;border-radius:15px;color:#fff;font-size:11px;min-width:125px;backdrop-filter:blur(8px)}.count b{font-size:31px;line-height:1.1;display:block}.layout{display:grid;grid-template-columns:350px minmax(0,1fr);gap:22px;align-items:start}.panel,.note{background:var(--card);border:1px solid var(--line);border-radius:18px;box-shadow:var(--shadow)}.panel{padding:23px;position:sticky;top:96px}.panel h2{font-size:20px;letter-spacing:-.04em;margin:0 0 2px}.panel:before{content:'Write it, snap it, keep it.';display:block;color:var(--muted);font-size:12px;margin-bottom:19px}.field{margin:13px 0}.field label{display:block;font-size:11px;font-weight:800;color:#444;margin:0 0 6px}.field input,.field textarea,.search{width:100%;border:1px solid #e4e4e4;border-radius:11px;padding:11px 12px;font:inherit;font-size:13px;background:#fbfbfb;outline:0;transition:border .18s,box-shadow .18s}.field input:focus,.field textarea:focus,.search:focus{border-color:#555;box-shadow:0 0 0 4px #11111110}textarea{min-height:120px;resize:vertical}.btn{border:0;border-radius:11px;background:var(--black);color:#fff;padding:12px 16px;font:inherit;font-size:13px;font-weight:800;cursor:pointer;box-shadow:0 7px 15px #0002;transition:transform .15s,box-shadow .15s}.btn:hover{transform:translateY(-1px);background:#303030;box-shadow:0 10px 20px #0003}.wide{width:100%}.tools{display:flex;gap:9px;align-items:center;margin-bottom:14px}.search{flex:1;min-width:100px;background:#fff}.export{color:#242424;font-weight:800;text-decoration:none;padding:9px 12px;border:1px solid #e2e2e2;border-radius:10px;background:#fff;font-size:11px;white-space:nowrap;cursor:pointer}.export:hover{background:#f1f1f1;border-color:#c8c8c8}.cards{display:grid;gap:13px}.note{padding:20px 22px;transition:transform .18s,box-shadow .18s}.note:hover{transform:translateY(-2px);box-shadow:0 20px 45px #00000012}.notehead{display:flex;justify-content:space-between;gap:12px}.note h3{font-size:16px;letter-spacing:-.03em;margin:0 0 6px}.subject{color:#282828;font-size:10px;font-weight:800;display:inline-flex;background:#f0f0f0;padding:4px 9px;border-radius:20px}.date{font-size:10px;color:var(--muted);white-space:nowrap}.content{white-space:pre-wrap;color:#303030;background:#fafafa;border:1px solid #ededed;border-radius:12px;padding:15px 17px;margin:8px 0 13px;font-size:13px;line-height:1.8;overflow-wrap:anywhere}.content-label{font-size:9px;text-transform:uppercase;letter-spacing:.13em;color:#858585;font-weight:800;margin-top:17px}.summary{background:#f6f6f6;border:1px solid #e8e8e8;border-left:3px solid #525252;border-radius:0 11px 11px 0;padding:12px 14px;margin:13px 0;white-space:pre-wrap;font-size:12px}.summarize{background:none;border:0;color:#242424;font:inherit;font-size:11px;font-weight:800;padding:6px 0;cursor:pointer}.summary-empty{color:var(--muted);font-style:italic}.summarize:hover{text-decoration:underline}.delete-note{float:right;background:#fff;border:1px solid #e9dede;border-radius:9px;color:#9d3d3d;padding:6px 10px;font:inherit;font-size:11px;font-weight:800;cursor:pointer}.delete-note:hover{background:#fff5f5}.toast{min-height:22px;color:#333;font-size:11px;margin-top:8px}.empty{text-align:center;background:#fff;border:1px dashed #d5d5d5;padding:48px 20px;border-radius:16px;color:var(--muted)}.empty b{display:block;color:var(--ink);font-size:19px;margin:9px}.empty:before{content:'✦';display:grid;place-items:center;margin:0 auto 12px;width:40px;height:40px;border-radius:13px;background:#ededed;color:#222}.photo-actions{display:flex;gap:8px;flex-wrap:wrap}.photo-preview{display:none;max-width:100%;max-height:220px;object-fit:contain;border-radius:12px;margin-top:10px}.note-photo{display:block;max-width:100%;max-height:420px;object-fit:contain;border-radius:13px;margin:12px 0;border:1px solid var(--line)}.photo-hint{display:block;color:var(--muted);font-size:10px;margin-top:6px}@media(max-width:760px){header{height:64px;padding:0 17px}.wrap{margin:22px auto;padding:0 15px}.hero{min-height:174px;padding:26px 23px;border-radius:18px;gap:12px}.hero h1{font-size:34px}.hero p{max-width:255px;font-size:12px}.count{padding:11px 13px;min-width:84px}.count b{font-size:25px}.layout{grid-template-columns:1fr;gap:16px}.panel{position:static;padding:19px}.tools{flex-wrap:wrap}.search{flex-basis:100%;order:-1}.note{padding:17px}.status{font-size:0;padding:8px}.status:first-letter{font-size:12px}}@media(max-width:410px){.hero{align-items:flex-start;padding:22px 17px}.hero h1{font-size:29px}.count{min-width:70px;padding:9px}.tools .export{font-size:10px;padding:8px}.notehead{align-items:flex-start}.date{font-size:9px}}@media(prefers-reduced-motion:reduce){*{scroll-behavior:auto!important;transition:none!important}}@media print{header,.panel,.tools,.summarize,.delete-note{display:none!important}body{background:#fff}.wrap{max-width:none;margin:0;padding:0}.layout{display:block}.hero{background:#fff;color:#111;padding:0;box-shadow:none;min-height:auto}.hero p,.eyebrow{color:#555}.count{color:#111;border-color:#ddd}.note{box-shadow:none;break-inside:avoid}.note-photo{max-height:8in}}
<style id="dark-theme">:root{color-scheme:dark;--ink:#f4f4f4;--muted:#a4a4a4;--paper:#080808;--card:#111;--line:#292929;--black:#f1f1f1;--black-soft:#202020;--mist:#171717;--shadow:0 16px 44px #0009}body{background:#080808;color:#f4f4f4}header{background:#0c0c0c;border-color:#252525}.brand{color:#f4f4f4}.logo{background:#f2f2f2;color:#111;box-shadow:none}.status{background:#171717;border-color:#303030;color:#d0d0d0}.wrap{color:#f4f4f4}.hero{background:linear-gradient(115deg,#050505,#161616 62%,#202020);border:1px solid #2a2a2a;box-shadow:0 18px 40px #0009}.hero:after{color:#ffffff08}.eyebrow,.hero p{color:#c3c3c3}.count{background:#202020;border-color:#3a3a3a;color:#fff}.panel,.note{background:#111;border-color:#292929;box-shadow:0 12px 35px #0008}.panel:before,.photo-hint{color:#a4a4a4}.field label{color:#e1e1e1}.field input,.field textarea,.search{background:#191919;border-color:#343434;color:#f5f5f5}.field input::placeholder,.field textarea::placeholder,.search::placeholder{color:#858585}.field input:focus,.field textarea:focus,.search:focus{border-color:#888;box-shadow:0 0 0 3px #ffffff12}.btn{background:#f1f1f1;color:#111;box-shadow:none}.btn:hover{background:#d8d8d8}.export{background:#191919;border-color:#383838;color:#ededed}.export:hover{background:#262626;border-color:#555}.note h3{color:#f5f5f5}.subject{background:#252525;color:#e8e8e8}.date,.content-label{color:#aaa}.content{background:#191919;border-color:#303030;color:#eee}.summary{background:#191919;border-color:#303030;border-left-color:#aaa;color:#eee}.summary-empty{color:#aaa}.summarize{color:#eee}.delete-note{background:#171717;border-color:#533535;color:#ff9a9a}.delete-note:hover{background:#291818}.toast{color:#ddd}.empty{background:#111;border-color:#383838;color:#aaa}.empty b{color:#f5f5f5}.empty:before{background:#252525;color:#eee}.photo-preview,.note-photo{background:#111;border-color:#303030}@media(max-width:760px){header{background:#0c0c0c}.status{color:#ddd}}@media print{body,.hero,.panel,.note,.content,.summary{background:#fff!important;color:#111!important}.hero p,.eyebrow{color:#444!important}.count{background:#fff!important;color:#111!important}}<style id="green-accent">:root{--green:#82e0a0;--green-bright:#a3f2b8;--ink:#f4f6f4;--muted:#9ba49d;--paper:#090c0a;--card:#111613;--line:#242d27;--shadow:0 20px 55px #0008}body{background:radial-gradient(ellipse at 50% -18%,#193322 0,transparent 42%),#090c0a;color:#f3f6f3;font-family:Inter,Manrope,system-ui,sans-serif}header{height:72px;background:#0c100deF;border-color:#222b25;backdrop-filter:blur(16px)}.brand{color:#f3f6f3;font-weight:800}.logo{background:linear-gradient(145deg,#a3f2b8,#59bc7c);color:#08110b;border-radius:12px;box-shadow:0 4px 18px #64d68a38}.status{background:#132018;border-color:#274532;color:#a7e9b5}.wrap{max-width:1200px;margin:34px auto}.hero{min-height:225px;padding:38px 44px;border-radius:24px;background:radial-gradient(ellipse at 84% 10%,#264a34 0,transparent 38%),linear-gradient(115deg,#101612,#121b15 62%,#17231a);border:1px solid #2a3a2e;box-shadow:0 24px 60px #0007}.hero:after{content:✳;right:13%;top:-86px;font-size:280px;color:#b3ffc10a}.eyebrow{color:#8be0a2;letter-spacing:.2em}.hero h1{font-size:clamp(35px,4.4vw,52px);color:#f6f8f6;letter-spacing:-.055em}.hero h1 span{color:#94e7a9}.hero p{color:#b0bbb3}.count{background:#111a14;border-color:#34483a;color:#f4f8f4}.count b{color:#a1edb3}.layout{grid-template-columns:350px minmax(0,1fr);gap:24px}.panel,.note{background:linear-gradient(155deg,#131915,#101411);border:1px solid #29332c;border-radius:20px;box-shadow:0 14px 38px #0005}.panel{padding:25px;top:92px}.panel h2{font-size:21px;color:#f4f7f4;letter-spacing:-.04em}.panel:before{color:#9ba79e;margin-bottom:22px}.field{margin:15px 0}.field label{color:#c8d1ca;font-size:11px;letter-spacing:.025em}.field input,.field textarea,.search{background:#0b0f0c;border-color:#2b352e;color:#f4f6f4;border-radius:12px}.field input::placeholder,.field textarea::placeholder,.search::placeholder{color:#727c74}.field input:focus,.field textarea:focus,.search:focus{border-color:#69c982;box-shadow:0 0 0 4px #63d07d16}textarea{min-height:130px}.btn{background:linear-gradient(110deg,#9bebae,#69ce88);color:#0b160e;border-radius:12px;font-weight:800;box-shadow:0 7px 20px #3c995233}.btn:hover{background:#a9f4b8}.photo-actions{gap:9px}.photo-actions .export{background:#121a15;border-color:#385241;color:#a5eab3}.photo-actions .export:hover{background:#19251c;border-color:#68b77c}.tools .export{background:#131814;border-color:#2b352e;color:#d4ded6}.tools .export:hover{background:#1a221c;border-color:#4a6551}.search{background:#111613}.cards{gap:15px}.note{padding:22px 24px;transition:transform .2s,border-color .2s,box-shadow .2s}.note:hover{transform:translateY(-2px);border-color:#405b47;box-shadow:0 20px 50px #0008}.note h3{color:#f2f6f2;font-size:17px;letter-spacing:-.035em}.subject{background:#18271d;color:#a4eab3;border:1px solid #2f4a36;padding:4px 10px}.date{color:#8e9a91}.content-label{color:#8ca195;letter-spacing:.15em}.content{background:#0c100d;border-color:#27312a;color:#fff;border-radius:13px;padding:16px 18px;font-size:13px;line-height:1.85}.summary{background:#131a15;border-color:#2b3a2f;border-left:3px solid #77d691;border-radius:0 13px 13px 0;color:#fff;padding:14px 16px;line-height:1.75}.summary b{color:#a5e9b4}.summary-empty{color:#a4afa6}.summarize{color:#95e4a8}.delete-note{background:#171312;border-color:#49312e;color:#f0a69c}.toast{color:#a4e9b2}.empty{background:#111613;border-color:#354239;color:#a8b2aa}.empty b{color:#f4f6f4}.empty:before{background:#18271d;color:#8fe4a4}.photo-hint{color:#98a49b}.photo-preview,.note-photo{border-color:#303b33}.photo-preview{border-radius:13px}@media(max-width:760px){header{height:64px}.wrap{margin:22px auto}.hero{min-height:180px;padding:26px 23px;border-radius:20px}.hero h1{font-size:34px}.layout{grid-template-columns:1fr}.panel{position:static;padding:20px}.note{padding:18px}.tools .export{font-size:10px}.status{color:#a7e9b5}}@media(max-width:410px){.hero{padding:22px 17px}.hero h1{font-size:30px}.note{padding:16px}.tools{gap:7px}}@media(prefers-reduced-motion:reduce){*{transition:none!important}}@media print{body,.hero,.panel,.note,.content,.summary{background:#fff!important;color:#111!important}.hero h1,.hero h1 span,.hero p,.eyebrow,.panel h2,.note h3,.content-label,.field label,.summary b{color:#111!important}}<style id="upload-feedback">.toast{scroll-margin-top:90px}.toast.error{display:block;margin-top:12px;padding:12px 14px;border-radius:12px;border:1px solid #75403a;background:#2a1514;color:#ffc1bb;font-size:13px;font-weight:700;line-height:1.55}.remove-attachment{margin-top:8px;padding:8px 11px;border:1px solid #69413d;border-radius:10px;background:#201414;color:#ffb8b1;font:inherit;font-size:11px;font-weight:800;cursor:pointer}.remove-attachment:hover{background:#311b19}.remove-attachment:not([hidden]){display:inline-flex;align-items:center}.btn:disabled{opacity:.65;cursor:wait;transform:none}@media(max-width:760px){.toast.error{position:relative;scroll-margin-top:75px}}<style id="mobile-save">html,body{min-height:100%;overflow-y:auto;-webkit-overflow-scrolling:touch}@media(max-width:760px){body.capture-visible #form>.btn.wide{position:fixed;left:16px;right:16px;bottom:calc(12px + env(safe-area-inset-bottom));width:calc(100vw - 32px);z-index:30;box-shadow:0 8px 26px #0009}body.capture-visible .wrap{padding-bottom:104px}body.capture-visible .toast.error{margin-bottom:78px}}<style id="note-accordion">.note-main{min-width:0;flex:1}.note-actions{display:flex;align-items:center;gap:9px;flex-shrink:0}.note-toggle{padding:8px 10px;font-size:10px}.note-snippet{margin-top:11px;color:#aab4ac;font-size:12px;line-height:1.6;display:-webkit-box;-webkit-line-clamp:2;-webkit-box-orient:vertical;overflow:hidden}.note-details{margin-top:14px;padding-top:4px;border-top:1px solid #28302a}.note-details[hidden]{display:none!important}.note-actions-bottom{display:flex;justify-content:space-between;align-items:center;gap:12px;margin-top:8px}.note-actions-bottom .delete-note{float:none}@media(max-width:760px){.tools #toggle-notes{flex:1}.notehead{align-items:flex-start}.note-actions{align-items:flex-end;flex-direction:column;gap:6px}.note-toggle{padding:8px 9px}}@media(max-width:410px){.note-actions{gap:5px}.note-toggle{font-size:9px;padding:7px 8px}.note-snippet{font-size:11px}}</style><header><div class="brand"><span class="logo">M</span> MemoryBox</div><div class="status">● &nbsp;Connected to your notebook</div></header><main class="wrap"><section class="hero"><div><div class="eyebrow">Your study space</div><h1>Keep what you <span>learn.</span></h1><p>Notes saved here stay in your PostgreSQL notebook.</p></div><div class="count"><b id="total">—</b>notes saved</div></section><section class="layout"><form class="panel" id="form"><h2>Capture a note</h2><div class="field"><label for="title">Title</label><input id="title" required placeholder="e.g. Cell respiration"></div><div class="field"><label for="subject">Subject</label><input id="subject" placeholder="e.g. Biology"></div><div class="field"><label for="content">Your notes</label><textarea id="content" placeholder="Write or paste what you want to remember… or add a photo or PDF below"></textarea></div><div class="field"><label>Attach notes</label><div class="photo-actions"><button class="export" type="button" onclick="document.querySelector('#camera').click()">Take a photo</button><button class="export" type="button" onclick="document.querySelector('#photo').click()">Choose photo/PDF</button></div><input id="photo" type="file" accept="image/*,application/pdf,.pdf" hidden><input id="camera" type="file" accept="image/*" capture="environment" hidden><small class="photo-hint">Photo AI checks that an image contains study material, reads handwriting, and prepares the text for a summary. Unrelated photos are not saved.</small><div id="attachment-name" class="photo-hint"></div><button id="remove-attachment" class="remove-attachment" type="button" hidden>Remove attachment ×</button><img id="photo-preview" class="photo-preview" alt="Selected note photo"></div><button class="btn wide">Save note&nbsp; ↗</button><div class="toast" id="toast" role="alert" aria-live="assertive"></div></form><div><div class="tools"><input class="search" id="q" placeholder="⌕  Search your notes…"><button class="export" id="toggle-notes" type="button" onclick="toggleAllNotes()">Expand all</button><a class="export" href="/export?format=markdown">Export ↗</a><button class="export" type="button" onclick="window.print()">Print / Save PDF</button></div><div class="cards" id="notes"></div></div></section></main><script src="https://cdnjs.cloudflare.com/ajax/libs/pdf.js/3.11.174/pdf.min.js"></script><script src="https://cdn.jsdelivr.net/npm/tesseract.js@5/dist/tesseract.min.js"></script><script>
if(window.pdfjsLib)pdfjsLib.GlobalWorkerOptions.workerSrc="https://cdnjs.cloudflare.com/ajax/libs/pdf.js/3.11.174/pdf.worker.min.js";
const esc=s=>String(s||'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
let expandedNotes=new Set();
function updateToggleAll(){let all=[...document.querySelectorAll(".note-details")],button=document.querySelector("#toggle-notes");if(button)button.textContent=all.length&&all.every(x=>!x.hidden)?"Collapse all":"Expand all"}
function toggleNote(id,button){let key=Number(id),details=document.querySelector("#note-details-"+key);if(!details)return;details.hidden=!details.hidden;if(details.hidden)expandedNotes.delete(key);else expandedNotes.add(key);button.textContent=details.hidden?"＋ Show note":"− Hide note";button.setAttribute("aria-expanded",String(!details.hidden));updateToggleAll()}
function toggleAllNotes(){let details=[...document.querySelectorAll(".note-details")],expand=details.some(x=>x.hidden);for(let item of details){item.hidden=!expand;let id=Number(item.id.replace("note-details-","")),button=document.querySelector("#note-toggle-"+id);if(expand)expandedNotes.add(id);else expandedNotes.delete(id);if(button){button.textContent=expand?"− Hide note":"＋ Show note";button.setAttribute("aria-expanded",String(expand))}}updateToggleAll()}
async function load(){let r=await fetch('/notes?q='+encodeURIComponent(document.querySelector('#q').value));if(!r.ok)throw Error('Could not load notes');let ns=await r.json();document.querySelector('#total').textContent=ns.length;document.querySelector('#notes').innerHTML=ns.length?ns.map(n=>{let id=Number(n.id),expanded=expandedNotes.has(id),plain=String(n.content||'').replace(/\s+/g,' ').trim(),snippet=plain.slice(0,150),shortened=plain.length>150;return "<article class=\"note\"><div class=\"notehead\"><div class=\"note-main\"><h3>"+esc(n.title)+"</h3><div class=\"subject\">"+esc(n.subject||'Unsorted')+"</div></div><div class=\"note-actions\"><time class=\"date\">"+new Date(n.created_at).toLocaleDateString()+"</time><button id=\"note-toggle-"+id+"\" class=\"note-toggle export\" aria-controls=\"note-details-"+id+"\" aria-expanded=\""+expanded+"\" onclick=\"toggleNote("+id+",this)\">"+(expanded?'− Hide note':'＋ Show note')+"</button></div></div><div class=\"note-snippet\">"+esc(snippet)+(shortened?'…':'')+"</div><div class=\"note-details\" id=\"note-details-"+id+"\""+(expanded?'':' hidden')+">"+(n.has_image?(n.image_type==='application/pdf'?"<a class=\"export\" target=\"_blank\" rel=\"noopener\" href=\"/notes/"+id+"/image\">Open attached PDF ↗</a>":"<img class=\"note-photo\" alt=\"Photo attached to note\" loading=\"lazy\" src=\"/notes/"+id+"/image\">"):"")+"<div class=\"content-label\">Saved text from your note</div><div class=\"content\">"+esc(n.content)+"</div><section class=\"summary\"><b>Student-friendly explanation</b><br>"+(n.summary?esc(n.summary):"<span class=\"summary-empty\">Simple explanation, key points, why it matters, and a quick recall question.</span>")+"</section><div class=\"note-actions-bottom\"><button class=\"summarize\" onclick=\"summarize("+id+")\">"+(n.summary?'↻ Refresh explanation':'✳ Explain this note')+"</button><button class=\"delete-note\" onclick=\"deleteNote("+id+")\">Delete</button></div></div></article>"}).join(''):'<div class=\"empty\">✎<b>Your notebook starts here</b>Save your first note using the form.</div>';updateToggleAll()}
async function deleteNote(id){if(!confirm('Delete this note permanently?'))return;let t=document.querySelector('#toast');t.textContent='Deleting note…';let r=await fetch('/notes/'+id,{method:'DELETE'});if(!r.ok){t.textContent=r.status===404?'This note was already deleted.':'Could not delete note. Please try again.';return}t.textContent='Note deleted.';expandedNotes.delete(id);load()}
async function summarize(id){let t=document.querySelector('#toast');t.textContent='Creating summary…';let r=await fetch('/notes/'+id+'/summarize',{method:'POST'});if(!r.ok){let message=await r.text();t.textContent=message||'AI summary is unavailable. Try again later.';load();return}t.textContent='Summary saved.';load()}
function preprocessForOCR(source){let width=source.width||source.naturalWidth,height=source.height||source.naturalHeight;if(!width||!height)return source;let scale=Math.min(2,2800/Math.max(width,height)),canvas=document.createElement('canvas');canvas.width=Math.max(1,Math.round(width*scale));canvas.height=Math.max(1,Math.round(height*scale));let ctx=canvas.getContext('2d',{willReadFrequently:true});ctx.drawImage(source,0,0,canvas.width,canvas.height);let frame=ctx.getImageData(0,0,canvas.width,canvas.height),pixels=frame.data,hist=new Uint32Array(256);for(let i=0;i<pixels.length;i+=4){let y=Math.round(.299*pixels[i]+.587*pixels[i+1]+.114*pixels[i+2]);hist[y]++}let total=pixels.length/4,low=0,high=255,sum=0;for(let i=0;i<256;i++){sum+=hist[i];if(sum>total*.04){low=i;break}}sum=0;for(let i=255;i>=0;i--){sum+=hist[i];if(sum>total*.04){high=i;break}}let span=Math.max(40,high-low);for(let i=0;i<pixels.length;i+=4){let y=Math.round(.299*pixels[i]+.587*pixels[i+1]+.114*pixels[i+2]);y=Math.max(0,Math.min(255,Math.round((y-low)*255/span)));pixels[i]=pixels[i+1]=pixels[i+2]=y}ctx.putImageData(frame,0,0);return canvas}
async function recognizeImage(source,logger,quick=false){let input=source,bitmap=null;try{if(typeof createImageBitmap==="function"&&!(source instanceof HTMLCanvasElement)){bitmap=await createImageBitmap(source);input=source instanceof HTMLCanvasElement?source:bitmap}input=preprocessForOCR(input)}catch(_){input=source}let worker;try{worker=await Tesseract.createWorker("eng",1,{logger});let candidates=[];let modes=quick?["3"]:["3","6"];for(let psm of modes){await worker.setParameters({tessedit_pageseg_mode:psm,preserve_interword_spaces:"1"});let result=await worker.recognize(input),data=result.data,text=(data.text||"").trim(),words=(text.match(/[A-Za-z0-9]{2,}/g)||[]).length;candidates.push({data,text,score:Number(data.confidence||0)+Math.min(words,240)*.04});if(psm==="3"&&Number(data.confidence||0)>=62&&text.length>=45)break}let best=candidates.sort((a,b)=>b.score-a.score)[0];if(!quick&&Number(best.data.confidence||0)<52){await worker.setParameters({tessedit_pageseg_mode:"11",preserve_interword_spaces:"1"});let result=await worker.recognize(input);if(Number(result.data.confidence||0)>Number(best.data.confidence||0))best={data:result.data,text:(result.data.text||"").trim()}}return{data:{text:best.text,confidence:Number(best.data.confidence||0)}}}finally{if(worker)await worker.terminate();if(bitmap)bitmap.close()}}
async function optimizePhonePhoto(file,status){status.textContent="Optimizing photo for a quicker upload…";let source,objectURL="";try{if(typeof createImageBitmap==="function")source=await createImageBitmap(file);else{objectURL=URL.createObjectURL(file);source=await new Promise((resolve,reject)=>{let img=new Image();img.onload=()=>resolve(img);img.onerror=()=>reject(Error("This photo format cannot be opened by your browser."));img.src=objectURL})}let width=source.width||source.naturalWidth,height=source.height||source.naturalHeight,scale=Math.min(1,2000/Math.max(width,height)),canvas=document.createElement("canvas");canvas.width=Math.max(1,Math.round(width*scale));canvas.height=Math.max(1,Math.round(height*scale));canvas.getContext("2d").drawImage(source,0,0,canvas.width,canvas.height);let blob=await new Promise(resolve=>canvas.toBlob(resolve,"image/jpeg",.82));canvas.width=canvas.height=0;if(!blob)throw Error("Could not prepare this photo.");return new File([blob],file.name.replace(/\.[^.]+$/,"")+".jpg",{type:"image/jpeg",lastModified:Date.now()})}finally{if(source&&typeof source.close==="function")source.close();if(objectURL)URL.revokeObjectURL(objectURL)}}function pdfTextWithLayout(items,pageWidth){let rows=[];for(let item of items){let text=(item.str||"").trim();if(!text)continue;let y=Array.isArray(item.transform)?item.transform[5]:null,x=Array.isArray(item.transform)?item.transform[4]:0,row=Number.isFinite(y)?rows.findLast?.(r=>Math.abs(r.y-y)<3):null;if(!row&&Number.isFinite(y)){for(let j=rows.length-1;j>=Math.max(0,rows.length-8);j--)if(Math.abs(rows[j].y-y)<3){row=rows[j];break}}if(!row){row={y:Number.isFinite(y)?y:rows.length,items:[]};rows.push(row)}row.items.push({x,width:Number(item.width||0),text})}rows.sort((a,b)=>b.y-a.y);let width=Number(pageWidth)||600,gapLimit=width*.05,splitRows=0;function segments(row){let sorted=row.items.sort((a,b)=>a.x-b.x),out=[],current=[];for(let item of sorted){let previous=current[current.length-1];if(previous&&item.x-(previous.x+previous.width)>gapLimit){out.push(current);current=[]}current.push(item)}if(current.length)out.push(current);return out}let segmented=rows.map(row=>segments(row));splitRows=segmented.filter(row=>row.length>1).length;if(splitRows<2)return rows.map(row=>row.items.sort((a,b)=>a.x-b.x).map(item=>item.text).join(" ").replace(/[ \t]+/g," ").trim()).filter(Boolean).join("\n");let spanning=[],left=[],right=[];for(let row of segmented){if(row.length>1){left.push(row[0].map(x=>x.text).join(" ").trim());right.push(row.slice(1).flat().map(x=>x.text).join(" ").trim());continue}let segment=row[0]||[],line=segment.map(x=>x.text).join(" ").trim();if(!line)continue;let start=segment[0].x,end=Math.max(...segment.map(x=>x.x+x.width));if(start<width*.12&&end>width*.82)spanning.push(line);else if(start<width*.5)left.push(line);else right.push(line)}return[spanning,left,right].filter(group=>group.length).map(group=>group.join("\n")).join("\n\n")}
async function extractNoteFile(file,t){let isPDF=file.type==='application/pdf'||file.name.toLowerCase().endsWith('.pdf');if(isPDF){if(!window.pdfjsLib)throw Error('PDF reader did not load. Refresh while connected to the internet.');let task=pdfjsLib.getDocument({data:new Uint8Array(await file.arrayBuffer())}),pdf=await task.promise;if(pdf.numPages>20)throw Error('This upload has '+pdf.numPages+' pages; the limit is 20 pages.');let pages=[],confidenceTotal=0,ocrPages=0;for(let i=1;i<=pdf.numPages;i++){t.textContent='Reading PDF page '+i+' of '+pdf.numPages+'…';let page=await pdf.getPage(i),base=page.getViewport({scale:1}),items=(await page.getTextContent()).items,text=pdfTextWithLayout(items,base.width);if(text.length<25){if(!window.Tesseract)throw Error('OCR reader did not load. Refresh while connected to the internet.');let scale=Math.min(1.5,3500/Math.max(base.width,base.height)),viewport=page.getViewport({scale}),canvas=document.createElement('canvas');canvas.width=Math.ceil(viewport.width);canvas.height=Math.ceil(viewport.height);await page.render({canvasContext:canvas.getContext('2d'),viewport}).promise;let result=await recognizeImage(canvas,m=>{if(m.status==='recognizing text')t.textContent='Reading scanned page '+i+' of '+pdf.numPages+'… '+Math.round(m.progress*100)+'%'});text=result.data.text.trim();confidenceTotal+=Number(result.data.confidence||0);ocrPages++;canvas.width=canvas.height=0}pages.push('Page '+i+'\n'+text)}return{text:pages.join('\n\n').trim().slice(0,60000),confidence:ocrPages?confidenceTotal/ocrPages:100}}if(!window.Tesseract)throw Error('Photo text reader did not load. Refresh while connected to the internet.');let result=await recognizeImage(file,m=>{if(m.status==='recognizing text')t.textContent='Quick photo scan… '+Math.round(m.progress*100)+'%'} ,true);return{text:result.data.text.trim(),confidence:Number(result.data.confidence||0)}}
let selectedAttachment=null,attachmentTextReady=false,ocrConfidence=100,originalOCRText="",previewURL=null,attachmentGeneration=0;
function showUploadError(status,message){status.textContent=message;status.classList.add("error");status.setAttribute("role","alert");status.scrollIntoView({behavior:"smooth",block:"center"})}
function resetAttachment(){attachmentGeneration++;selectedAttachment=null;attachmentTextReady=false;ocrConfidence=100;originalOCRText="";document.querySelector("#photo").value="";document.querySelector("#camera").value="";document.querySelector("#attachment-name").textContent="";document.querySelector("#remove-attachment").hidden=true;let p=document.querySelector("#photo-preview");p.style.display="none";p.removeAttribute("src");if(previewURL){URL.revokeObjectURL(previewURL);previewURL=null}let save=document.querySelector("#form .btn");save.disabled=false;save.textContent="Save note ↗"}
function removeAttachment(){resetAttachment();document.querySelector("#content").value="";let status=document.querySelector("#toast");status.classList.remove("error");status.textContent="Attachment removed. You can choose another file or write a note."}
async function chooseAttachment(input){let file=input.files[0],preview=document.querySelector("#photo-preview"),status=document.querySelector("#toast"),save=document.querySelector("#form .btn");if(!file)return;let run=++attachmentGeneration;status.classList.remove("error");selectedAttachment=file;attachmentTextReady=false;ocrConfidence=100;originalOCRText="";document.querySelector("#content").value="";(input.id==="camera"?document.querySelector("#photo"):document.querySelector("#camera")).value="";document.querySelector("#attachment-name").textContent=file.name+" selected";document.querySelector("#remove-attachment").hidden=false;if(previewURL){URL.revokeObjectURL(previewURL);previewURL=null}let isPDF=file.type==="application/pdf"||file.name.toLowerCase().endsWith(".pdf");preview.style.display=isPDF?"none":"block";save.disabled=true;save.textContent="Quick scan…";let progress={set textContent(value){if(run===attachmentGeneration)status.textContent=value}};try{if(!isPDF){selectedAttachment=await optimizePhonePhoto(file,progress);if(run!==attachmentGeneration)return;previewURL=URL.createObjectURL(selectedAttachment);preview.src=previewURL}let extracted=await extractNoteFile(isPDF?selectedAttachment:file,progress);if(run!==attachmentGeneration)return;ocrConfidence=extracted.confidence;originalOCRText=extracted.text;document.querySelector("#content").value=extracted.text;attachmentTextReady=true;if(!extracted.text){status.classList.remove("error");status.textContent=isPDF?"No text found in this PDF. Type the note text before saving it.":"Handwritten notes are okay. The photo AI will read them when you save; you can also type or correct the text here.";return}status.textContent=ocrConfidence<55?"Quick scan found text that may need correction. Review it, or let AI read the photo when you save.":"Quick scan ready. Review or correct the text, then tap Save photo + summary.";}catch(err){if(run!==attachmentGeneration)return;attachmentTextReady=true;let pdf=file.type==="application/pdf"||file.name.toLowerCase().endsWith(".pdf");let supportedImage=file.type.startsWith("image/")||/\.(heic|heif|avif)$/i.test(file.name);if(supportedImage){status.classList.remove("error");status.textContent="Handwriting is okay. Automatic scan struggled; tap Save and AI will try to read the photo, or type the notes here."}else showUploadError(status,pdf?"Could not read this PDF. Check that it is valid and not password-protected.":"This file is not a supported photo or PDF. Choose an image or a valid PDF.")}finally{if(run===attachmentGeneration){save.disabled=false;save.textContent="Save photo + summary ↗"}}}
document.querySelector("#photo").addEventListener("change",e=>chooseAttachment(e.target));document.querySelector("#camera").addEventListener("change",e=>chooseAttachment(e.target));document.querySelector("#remove-attachment").addEventListener("click",removeAttachment);
document.querySelector("#form").addEventListener("submit",async e=>{e.preventDefault();let status=document.querySelector("#toast"),form=e.currentTarget,attachment=selectedAttachment,titleValue=document.querySelector("#title").value.trim(),subjectValue=document.querySelector("#subject").value.trim(),contentValue=document.querySelector("#content").value.trim();status.classList.remove("error");if(!titleValue){showUploadError(status,"Add a title first.");return}if(attachment&&!attachmentTextReady){showUploadError(status,"Wait for the photo or PDF text reading to finish.");return}if(!contentValue&&(!attachment||!attachment.type.startsWith("image/"))){showUploadError(status,"Add readable text before saving this file.");return}let response,saveButton=form.querySelector(".btn");saveButton.disabled=true;try{if(attachment){status.textContent=attachment.type.startsWith("image/")?"Checking the photo and reading the notes…":"Checking whether this is study material…";let data=new FormData();data.append("title",titleValue);data.append("subject",subjectValue);data.append("content",contentValue);data.append("text_edited",String(contentValue.replace(/\s+/g," ").trim()!==originalOCRText.replace(/\s+/g," ").trim()));data.append("photo",attachment,attachment.name);response=await fetch("/notes/photo",{method:"POST",body:data})}else{status.textContent="Saving…";response=await fetch("/notes",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({title:titleValue,subject:subjectValue,content:contentValue})})}}catch(_){saveButton.disabled=false;showUploadError(status,"Could not reach MemoryBox. Check your connection and try again.");return}if(!response.ok){saveButton.disabled=false;let message=await response.text();showUploadError(status,response.status===413?"This file is too large for the server. Try a smaller photo or PDF.":(message||"Could not save this note."));return}let note=await response.json();expandedNotes.add(Number(note.id));form.reset();resetAttachment();status.classList.remove("error");status.textContent=attachment?"Photo and readable text saved. Creating your student-friendly explanation…":"Saved to your notebook.";load();if(attachment){fetch("/notes/"+note.id+"/summarize",{method:"POST"}).then(async result=>{if(!result.ok){let message=await result.text();status.textContent="Your file and text are saved. "+(message||"The explanation could not be created right now.");return}status.textContent="Photo, readable text, and explanation saved.";load()}).catch(()=>{status.textContent="Your file and text are saved. The explanation could not be created right now."})}});if('IntersectionObserver' in window){let captureObserver=new IntersectionObserver(entries=>document.body.classList.toggle('capture-visible',entries[0].isIntersecting),{threshold:0});captureObserver.observe(document.querySelector('#form'))}let timer;document.querySelector('#q').addEventListener('input',()=>{clearTimeout(timer);timer=setTimeout(load,180)});load().catch(()=>document.querySelector('#notes').innerHTML='<div class="empty">Could not connect to MemoryBox. Check that the server is running.</div>');
</script></html>`

func respond(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
func (a *App) create(w http.ResponseWriter, r *http.Request) {
	var n Note
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(&n); err != nil {
		http.Error(w, "invalid note JSON", 400)
		return
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		http.Error(w, "send exactly one JSON object", 400)
		return
	}
	n.Title = strings.TrimSpace(n.Title)
	n.Content = strings.TrimSpace(n.Content)
	n.Subject = strings.TrimSpace(n.Subject)
	if n.Title == "" || n.Content == "" {
		http.Error(w, "title and content are required", 400)
		return
	}
	err := a.DB.QueryRow(r.Context(), `INSERT INTO notes(title,subject,content) VALUES($1,$2,$3) RETURNING id,created_at`, n.Title, n.Subject, n.Content).Scan(&n.ID, &n.CreatedAt)
	if err != nil {
		log.Print(err)
		http.Error(w, "could not save note", 500)
		return
	}
	respond(w, 201, n)
}

func (a *App) createPhoto(w http.ResponseWriter, r *http.Request) {
	const maxPhotoSize = 20 << 20
	r.Body = http.MaxBytesReader(w, r.Body, maxPhotoSize+(1<<20))
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		http.Error(w, "invalid photo upload (maximum size is 20 MB)", http.StatusBadRequest)
		return
	}
	title := strings.TrimSpace(r.FormValue("title"))
	subject := strings.TrimSpace(r.FormValue("subject"))
	content := strings.TrimSpace(r.FormValue("content"))
	if title == "" {
		http.Error(w, "title is required", http.StatusBadRequest)
		return
	}
	file, fileHeader, err := r.FormFile("photo")
	if err != nil {
		http.Error(w, "select a photo to upload", http.StatusBadRequest)
		return
	}
	defer file.Close()
	imageData, err := io.ReadAll(io.LimitReader(file, maxPhotoSize+1))
	if err != nil || len(imageData) == 0 || len(imageData) > maxPhotoSize {
		http.Error(w, "photo must be between 1 byte and 20 MB", http.StatusBadRequest)
		return
	}
	imageType := http.DetectContentType(imageData[:min(len(imageData), 512)])
	if imageType == "application/octet-stream" {
		declaredType := strings.ToLower(strings.TrimSpace(fileHeader.Header.Get("Content-Type")))
		if declaredType == "image/heic" || declaredType == "image/heif" || declaredType == "image/avif" {
			imageType = declaredType
		}
	}
	if imageType != "application/pdf" && imageType != "image/jpeg" && imageType != "image/png" && imageType != "image/gif" && imageType != "image/webp" && imageType != "image/heic" && imageType != "image/heif" && imageType != "image/avif" {
		http.Error(w, "attachment must be a photo or PDF", http.StatusBadRequest)
		return
	}
	textEdited := r.FormValue("text_edited") == "true"
	transcribeImage := strings.HasPrefix(imageType, "image/") && !textEdited
	if imageType == "application/pdf" && content == "" {
		http.Error(w, "No readable text was found in this PDF. Upload a searchable PDF or type the note text; the file was not saved.", http.StatusUnprocessableEntity)
		return
	}
	checkCtx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()
	relevant, reason, transcription := false, "", ""
	var checkErr error
	relevant, reason, transcription, checkErr = checkStudyRelevance(checkCtx, content, imageType, imageData, transcribeImage)
	if checkErr != nil {
		log.Printf("attachment AI check failed: %v", checkErr)
		if imageType == "application/pdf" {
			http.Error(w, "Could not check this PDF right now. It was not saved; please try again shortly.", http.StatusServiceUnavailable)
			return
		}
		// If AI is rate-limited, allow only text that independently has clear
		// study signals. An unreadable or ordinary photo is not saved.
		relevant = hasStudyTextSignals(content)
		if !relevant {
			http.Error(w, "MemoryBox could not verify this photo as study material because its AI reader is temporarily busy. It was not saved; retry in a little while.", http.StatusServiceUnavailable)
			return
		}
	}
	if strings.TrimSpace(transcription) != "" {
		cleaned := strings.TrimSpace(transcription)
		if !strings.Contains(strings.ToLower(cleaned), "[no readable text]") {
			content = cleaned
		}
	}
	if content == "" {
		if strings.HasPrefix(imageType, "image/") {
			content = "[Handwritten note photo saved. The text could not be read automatically. Open the attached photo to read the original notes.]"
		} else {
			http.Error(w, "No readable text was found in this PDF. Please type the note text; the file was not saved.", http.StatusUnprocessableEntity)
			return
		}
	}
	content = cleanStudyText(content)
	if !relevant {
		kind := "photo"
		if imageType == "application/pdf" {
			kind = "PDF"
		}
		if reason == "" {
			reason = "it does not appear to contain study material"
		}
		http.Error(w, "This "+kind+" does not appear to be study-related: "+reason, http.StatusUnprocessableEntity)
		return
	}
	var n Note
	n.Title, n.Subject, n.Content, n.HasImage, n.ImageType = title, subject, content, true, imageType
	err = a.DB.QueryRow(r.Context(), `INSERT INTO notes(title,subject,content,image_data,image_type) VALUES($1,$2,$3,$4,$5) RETURNING id,created_at`, title, subject, content, imageData, imageType).Scan(&n.ID, &n.CreatedAt)
	if err != nil {
		log.Print(err)
		http.Error(w, "could not save note photo", http.StatusInternalServerError)
		return
	}
	respond(w, http.StatusCreated, n)
}

func cleanStudyText(text string) string {
	text = strings.ReplaceAll(text, "\x00", "")
	text = strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.Join(strings.Fields(line), " ")
	}
	text = strings.Join(lines, "\n")
	for strings.Contains(text, "\n\n\n") {
		text = strings.ReplaceAll(text, "\n\n\n", "\n\n")
	}
	return strings.TrimSpace(text)
}

func needsVisionTranscription(text string) bool {
	if strings.HasPrefix(text, "[Handwritten note photo saved.") {
		return true
	}
	lettersDigits, symbols, shortWords, words := 0, 0, 0, 0
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			lettersDigits++
		} else if !unicode.IsSpace(r) {
			symbols++
		}
	}
	for _, word := range strings.Fields(text) {
		words++
		if len([]rune(strings.Trim(word, ".,:;!?()[]{}'\""))) <= 2 {
			shortWords++
		}
	}
	return lettersDigits+symbols >= 60 && (float64(lettersDigits)/float64(lettersDigits+symbols) < .62 || (words >= 12 && float64(shortWords)/float64(words) > .42))
}

func (a *App) image(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		http.Error(w, "invalid note id", http.StatusBadRequest)
		return
	}
	var data []byte
	var contentType string
	if err := a.DB.QueryRow(r.Context(), `SELECT image_data,image_type FROM notes WHERE id=$1 AND image_data IS NOT NULL`, id).Scan(&data, &contentType); err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.Write(data)
}

func (a *App) delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		http.Error(w, "invalid note id", http.StatusBadRequest)
		return
	}
	tag, err := a.DB.Exec(r.Context(), `DELETE FROM notes WHERE id=$1`, id)
	if err != nil {
		log.Print(err)
		http.Error(w, "could not delete note", http.StatusInternalServerError)
		return
	}
	if tag.RowsAffected() == 0 {
		http.NotFound(w, r)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) find(r *http.Request) ([]Note, error) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	subject := r.URL.Query().Get("subject")
	query := `SELECT id,title,subject,content,summary,created_at,(image_data IS NOT NULL),image_type FROM notes WHERE ($1='' OR strpos(lower(title||' '||subject||' '||content||' '||summary),lower($1))>0) AND ($2='' OR subject=$2) ORDER BY id DESC`
	if r.URL.Path != "/export" {
		query += " LIMIT 200"
	}
	rows, err := a.DB.Query(r.Context(), query, q, subject)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	notes := []Note{}
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.Title, &n.Subject, &n.Content, &n.Summary, &n.CreatedAt, &n.HasImage, &n.ImageType); err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}
func (a *App) list(w http.ResponseWriter, r *http.Request) {
	n, err := a.find(r)
	if err != nil {
		http.Error(w, "could not load notes", 500)
		return
	}
	respond(w, 200, n)
}
func (a *App) export(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format != "" && format != "json" && format != "markdown" {
		http.Error(w, "format must be json or markdown", 400)
		return
	}
	notes, err := a.find(r)
	if err != nil {
		http.Error(w, "could not export notes", 500)
		return
	}
	if format == "markdown" {
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="memorybox.md"`)
		for _, n := range notes {
			fmt.Fprintf(w, "# %s\n\nSubject: %s\n\n%s\n\n", n.Title, n.Subject, n.Content)
			if n.Summary != "" {
				fmt.Fprintf(w, "## Summary\n\n%s\n\n", n.Summary)
			}
		}
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="memorybox.json"`)
	respond(w, 200, notes)
}
func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	db, err := pgxpool.New(ctx, env("DATABASE_URL", "postgres://memorybox:memorybox-local@localhost:55432/memorybox?sslmode=disable"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err = db.Ping(ctx); err != nil {
		log.Fatal(err)
	}
	if _, err = db.Exec(ctx, schema); err != nil {
		log.Fatal(err)
	}
	for _, migration := range []string{
		`ALTER TABLE notes ADD COLUMN IF NOT EXISTS image_data BYTEA`,
		`ALTER TABLE notes ADD COLUMN IF NOT EXISTS image_type TEXT NOT NULL DEFAULT ''`,
	} {
		if _, err = db.Exec(ctx, migration); err != nil {
			log.Fatal(err)
		}
	}
	if _, err = db.Exec(ctx, `UPDATE notes SET summary='' WHERE summary LIKE 'Quick backup summary (AI service temporarily unavailable)%'`); err != nil {
		log.Fatal(err)
	}
	addr := env("LISTEN_ADDR", "127.0.0.1:8084")
	log.Printf("MemoryBox listening on %s", addr)
	app := &App{DB: db}
	go app.runSwitch(context.Background())
	srv := &http.Server{Addr: addr, Handler: app.routes(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 60 * time.Second, WriteTimeout: 120 * time.Second, IdleTimeout: 60 * time.Second}
	log.Fatal(srv.ListenAndServe())
}
