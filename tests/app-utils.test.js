const test=require('node:test');
const assert=require('node:assert/strict');
const {escapeHTML,selectLocale,filterAndSortPosts,selectSortOrder,buildPostPayload,requestJSON}=require('../web/app-utils.js');

test('escapes HTML-sensitive characters',()=>{
 assert.equal(escapeHTML(`<a title="x">Tom & Jerry's</a>`),'&lt;a title=&quot;x&quot;&gt;Tom &amp; Jerry&#39;s&lt;/a&gt;');
});

test('converts non-string values before escaping HTML',()=>{
 assert.equal(escapeHTML(123), '123');
 assert.equal(escapeHTML(null), 'null');
});

test('selects Japanese from the browser language',()=>{
 assert.equal(selectLocale('', 'ja-JP'),'ja');
 assert.equal(selectLocale(null, 'JA'),'ja');
});

test('falls back to English when the browser language is unavailable or not Japanese',()=>{
 assert.equal(selectLocale('', ''),'en');
 assert.equal(selectLocale(null, 'fr-FR'),'en');
});

test('prefers the stored locale over the browser language',()=>{
 assert.equal(selectLocale('en','ja-JP'),'en');
 assert.equal(selectLocale('ja','en-US'),'ja');
});

const posts=[
 {title:'Beta',path:'posts/b.md',date:'2026-01-02',modified:'2026-01-03T00:00:00Z'},
 {title:'Alpha',path:'notes/a.md',date:'2026-01-03',modified:'2026-01-01T00:00:00Z'},
 {title:'Alpha Two',path:'posts/a.md',date:'',modified:'2026-01-03T00:00:00Z'}
];

test('filters posts by title and path without regard to case',()=>{
 assert.deepEqual(filterAndSortPosts(posts,'ALPHA','modified-desc').map(post=>post.path),['posts/a.md','notes/a.md']);
 assert.deepEqual(filterAndSortPosts(posts,'NOTES','modified-desc').map(post=>post.path),['notes/a.md']);
});

test('sorts posts by newest and oldest modification time',()=>{
 assert.deepEqual(filterAndSortPosts(posts,'','modified-desc').map(post=>post.path),['posts/a.md','posts/b.md','notes/a.md']);
 assert.deepEqual(filterAndSortPosts(posts,'','modified-asc').map(post=>post.path),['notes/a.md','posts/a.md','posts/b.md']);
});

test('sorts posts by newest publication date',()=>{
 assert.deepEqual(filterAndSortPosts(posts,'','date-desc').map(post=>post.path),['notes/a.md','posts/b.md','posts/a.md']);
});

test('does not mutate the original posts array while sorting',()=>{
 const originalOrder=posts.map(post=>post.path);
 filterAndSortPosts(posts,'','modified-desc');
 assert.deepEqual(posts.map(post=>post.path),originalOrder);
});

test('restores supported sort orders and rejects stale values',()=>{
 assert.equal(selectSortOrder('modified-desc'),'modified-desc');
 assert.equal(selectSortOrder('modified-asc'),'modified-asc');
 assert.equal(selectSortOrder('date-desc'),'date-desc');
 assert.equal(selectSortOrder('unknown-order'),'modified-desc');
 assert.equal(selectSortOrder(null),'modified-desc');
});

test('builds a save payload without changing Markdown whitespace',()=>{
 const payload=buildPostPayload({path:'posts/renamed.md',frontMatter:'title: Test',body:'\nFirst line'}, {path:'posts/original.md',delimiter:'---',modified:'stamp'});
 assert.deepEqual(payload,{path:'posts/renamed.md',originalPath:'posts/original.md',frontMatter:'title: Test',body:'\nFirst line',delimiter:'---',modified:'stamp'});
});

test('requests JSON with language and content headers',async()=>{
 let received;
 const result=await requestJSON(async(url,options)=>{received={url,options};return {ok:true,status:200,json:async()=>({value:1})}},'/api/example',{method:'POST',headers:{'X-Test':'yes'}},'ja');
 assert.deepEqual(result,{value:1});
 assert.equal(received.url,'/api/example');
 assert.deepEqual(received.options.headers,{'Content-Type':'application/json','Accept-Language':'ja','X-Test':'yes'});
});

test('returns null for a successful empty response',async()=>{
 assert.equal(await requestJSON(async()=>({ok:true,status:204}),'/api/empty'),null);
});

test('throws the API error message from a JSON response',async()=>{
 await assert.rejects(()=>requestJSON(async()=>({ok:false,status:400,statusText:'Bad Request',json:async()=>({error:'Invalid post'})}),'/api/post'),/Invalid post/);
});

test('falls back to the HTTP status text for a non-JSON error',async()=>{
 await assert.rejects(()=>requestJSON(async()=>({ok:false,status:500,statusText:'Server Error',json:async()=>{throw Error('not json')}}),'/api/post'),/Server Error/);
});
