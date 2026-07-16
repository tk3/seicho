const $=s=>document.querySelector(s);let posts=[],current=null,dirty=false,siteConfigured=false;
async function api(url,options={}){const r=await fetch(url,{headers:{'Content-Type':'application/json'},...options});if(!r.ok){let e=await r.json().catch(()=>({error:r.statusText}));throw Error(e.error)}return r.status===204?null:r.json()}
function showSetup(show=true){$('#setup').classList.toggle('hidden',!show);$('#close-setup').classList.toggle('hidden',!siteConfigured)}
async function boot(){const site=await api('/api/site');siteConfigured=site.configured;$('#app-version').textContent='v'+site.version;if(!site.configured)return showSetup();$('#site-name').textContent=site.path;await loadPosts()}
$('#site-form').onsubmit=async e=>{e.preventDefault();try{const site=await api('/api/site',{method:'POST',body:JSON.stringify({path:$('#site-path').value})});siteConfigured=true;$('#app-version').textContent='v'+site.version;$('#site-name').textContent=site.path;showSetup(false);$('#setup-error').value='';await loadPosts()}catch(e){$('#setup-error').value=e.message}}
$('#change-site').onclick=()=>showSetup();
$('#close-setup').onclick=()=>showSetup(false);
async function loadPosts(){posts=await api('/api/posts');renderList()}
function renderList(){const q=$('#search').value.toLowerCase(),sort=$('#sort').value;const compare={"modified-desc":(a,b)=>new Date(b.modified)-new Date(a.modified),"modified-asc":(a,b)=>new Date(a.modified)-new Date(b.modified),"date-desc":(a,b)=>(b.date||'').localeCompare(a.date||'')}[sort];$('#post-list').innerHTML='';posts.filter(p=>(p.title+' '+p.path).toLowerCase().includes(q)).sort((a,b)=>compare(a,b)||a.path.localeCompare(b.path)).forEach(p=>{const el=document.createElement('div');el.className='post'+(current?.path===p.path?' active':'');const updated=new Date(p.modified).toLocaleString('ja-JP',{year:'numeric',month:'2-digit',day:'2-digit',hour:'2-digit',minute:'2-digit'});el.innerHTML=`<strong>${esc(p.title)}${p.draft?'<span class="draft">DRAFT</span>':''}</strong><small>更新 ${esc(updated)}</small>`;el.onclick=()=>openPost(p.path);$('#post-list').append(el)})}
$('#search').oninput=renderList;
$('#sort').onchange=renderList;
async function openPost(path){if(dirty&&!confirm('保存していない変更を破棄しますか？'))return;current=await api('/api/post?path='+encodeURIComponent(path));$('#path').value=current.path;$('#front').value=current.frontMatter;$('#body').value=current.body;dirty=false;showEditor();render();renderList()}
function showEditor(){ $('#empty').classList.add('hidden');$('#editor').classList.remove('hidden') }
function showNewPost(show=true){$('#new-post-modal').classList.toggle('hidden',!show)}
function newPost(){if(dirty&&!confirm('保存していない変更を破棄しますか？'))return;$('#new-post-path').value='posts/new-post.md';$('#new-post-error').value='';showNewPost();setTimeout(()=>{$('#new-post-path').focus();$('#new-post-path').select()},0)}
$('#new-post-form').onsubmit=async e=>{e.preventDefault();const path=$('#new-post-path').value.trim();if(!path)return;try{current=await api('/api/post',{method:'POST',body:JSON.stringify({path})});showNewPost(false);$('#path').value=current.path;$('#front').value=current.frontMatter;$('#body').value=current.body;dirty=false;showEditor();render();await loadPosts();$('#body').focus();toast('Hugoのarchetypeから投稿を作成しました')}catch(e){$('#new-post-error').value=e.message}}
$('#close-new-post').onclick=$('#cancel-new-post').onclick=()=>showNewPost(false);
$('#new').onclick=$('#empty-new').onclick=newPost;
['path','front','body'].forEach(id=>$('#'+id).addEventListener('input',()=>{dirty=true;render()}));
let renderTimer=0,renderVersion=0;
function render(){clearTimeout(renderTimer);const version=++renderVersion;renderTimer=setTimeout(async()=>{try{const result=await api('/api/preview',{method:'POST',body:JSON.stringify({markdown:$('#body').value})});if(version===renderVersion)$('#preview').innerHTML=result.html||'<p style="color:#999">プレビューがここに表示されます。</p>'}catch(e){if(version===renderVersion)$('#preview').textContent='プレビューエラー: '+e.message}},120)}
function esc(s){return String(s).replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]))}
$('#save').onclick=save;
async function save(){if(!current)return;try{current=await api('/api/post',{method:'PUT',body:JSON.stringify({path:$('#path').value,originalPath:current.path,frontMatter:$('#front').value,body:$('#body').value,delimiter:current.delimiter,modified:current.modified})});$('#path').value=current.path;dirty=false;toast('保存しました');await loadPosts()}catch(e){toast(e.message,true)}}
function showDeletePost(show=true){$('#delete-post-modal').classList.toggle('hidden',!show)}
$('#delete').onclick=()=>{if(!current)return;$('#delete-post-path').textContent=current.path;$('#delete-post-error').value='';showDeletePost()}
$('#delete-post-form').onsubmit=async e=>{e.preventDefault();if(!current)return;try{await api('/api/post?path='+encodeURIComponent(current.path),{method:'DELETE'});showDeletePost(false);current=null;dirty=false;$('#editor').classList.add('hidden');$('#empty').classList.remove('hidden');await loadPosts();toast('削除しました')}catch(e){$('#delete-post-error').value=e.message}}
$('#close-delete-post').onclick=$('#cancel-delete-post').onclick=()=>showDeletePost(false);
document.addEventListener('keydown',e=>{if((e.ctrlKey||e.metaKey)&&e.key.toLowerCase()==='s'){e.preventDefault();save()}});window.addEventListener('beforeunload',e=>{if(dirty){e.preventDefault();e.returnValue=''}});
document.addEventListener('keydown',e=>{if(e.key!=='Escape')return;if(!$('#new-post-modal').classList.contains('hidden')){showNewPost(false);return}if(!$('#delete-post-modal').classList.contains('hidden')){showDeletePost(false);return}if(siteConfigured&&!$('#setup').classList.contains('hidden'))showSetup(false)});
function toast(msg,bad=false){const el=$('#toast');el.textContent=msg;el.style.background=bad?'#a93e2b':'';el.classList.add('show');setTimeout(()=>el.classList.remove('show'),2600)}
boot().catch(e=>toast(e.message,true));
