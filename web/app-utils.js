(function(root){
function escapeHTML(value){
 return String(value).replace(/[&<>"']/g,character=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[character]));
}
function selectLocale(storedLocale,browserLanguage){
 if(storedLocale)return storedLocale;
 return String(browserLanguage||'').toLowerCase().startsWith('ja')?'ja':'en';
}
function filterAndSortPosts(posts,query,sort){
 const normalizedQuery=String(query||'').toLowerCase();
 const compare={
  'modified-desc':(a,b)=>new Date(b.modified)-new Date(a.modified),
  'modified-asc':(a,b)=>new Date(a.modified)-new Date(b.modified),
  'date-desc':(a,b)=>(b.date||'').localeCompare(a.date||'')
 }[sort];
 return posts.filter(post=>(post.title+' '+post.path).toLowerCase().includes(normalizedQuery)).sort((a,b)=>compare(a,b)||a.path.localeCompare(b.path));
}
function buildPostPayload(fields,current){
 return {path:fields.path,originalPath:current.path,frontMatter:fields.frontMatter,body:fields.body,delimiter:current.delimiter,modified:current.modified};
}
async function requestJSON(fetchImplementation,url,options={},locale='en'){
 const response=await fetchImplementation(url,{...options,headers:{'Content-Type':'application/json','Accept-Language':locale,...options.headers}});
 if(!response.ok){
  const error=await response.json().catch(()=>({error:response.statusText}));
  throw Error(error.error);
 }
 return response.status===204?null:response.json();
}
function postRoute(path){
 return '/edit/'+String(path).split('/').map(segment=>encodeURIComponent(segment)).join('/');
}
function postPathFromRoute(pathname){
 if(!String(pathname).startsWith('/edit/'))return null;
 const encodedPath=String(pathname).slice('/edit/'.length);
 if(!encodedPath)return null;
 try{return encodedPath.split('/').map(segment=>decodeURIComponent(segment)).join('/')}
 catch(error){return null}
}
const api={escapeHTML,selectLocale,filterAndSortPosts,buildPostPayload,requestJSON,postRoute,postPathFromRoute};
if(typeof module==='object'&&module.exports)module.exports=api;
else Object.assign(root,api);
})(typeof globalThis==='object'?globalThis:this);
