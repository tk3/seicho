const test=require('node:test');
const assert=require('node:assert/strict');
const {postRoute,postPathFromRoute,replacePostRoute,clearPostRoute}=require('../web/router.js');

test('builds an editor route while preserving path hierarchy',()=>{
 assert.equal(postRoute('posts/hello world.md'),'/edit/posts/hello%20world.md');
 assert.equal(postRoute('記事/清張.md'),'/edit/%E8%A8%98%E4%BA%8B/%E6%B8%85%E5%BC%B5.md');
});

test('reads a post path from an editor route',()=>{
 assert.equal(postPathFromRoute('/edit/posts/hello%20world.md'),'posts/hello world.md');
 assert.equal(postPathFromRoute('/edit/%E8%A8%98%E4%BA%8B/%E6%B8%85%E5%BC%B5.md'),'記事/清張.md');
});

test('ignores non-editor and malformed routes',()=>{
 assert.equal(postPathFromRoute('/'),null);
 assert.equal(postPathFromRoute('/edit/'),null);
 assert.equal(postPathFromRoute('/edit/%E0%A4%A'),null);
});

test('replaces browser history for selected and cleared posts',()=>{
 const paths=[];
 const historyObject={replaceState(state,title,path){paths.push(path)}};
 replacePostRoute(historyObject,'posts/hello.md');
 clearPostRoute(historyObject);
 assert.deepEqual(paths,['/edit/posts/hello.md','/']);
});
