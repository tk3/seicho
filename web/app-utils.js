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
const sortOrders=['modified-desc','modified-asc','date-desc'];
function selectSortOrder(storedOrder){return sortOrders.includes(storedOrder)?storedOrder:'modified-desc'}
function buildPostPayload(fields,current){
 return {path:fields.path,originalPath:current.path,frontMatter:fields.frontMatter,body:fields.body,delimiter:current.delimiter,modified:current.modified};
}
function closeDetails(details){if(details)details.open=false}
async function requestJSON(fetchImplementation,url,options={},locale='en'){
 const response=await fetchImplementation(url,{...options,headers:{'Content-Type':'application/json','Accept-Language':locale,...options.headers}});
 if(!response.ok){
  const error=await response.json().catch(()=>({error:response.statusText}));
  throw Error(error.error);
 }
 return response.status===204?null:response.json();
}
const api={escapeHTML,selectLocale,filterAndSortPosts,selectSortOrder,buildPostPayload,closeDetails,requestJSON};
if(typeof module==='object'&&module.exports)module.exports=api;
else Object.assign(root,api);
})(typeof globalThis==='object'?globalThis:this);
