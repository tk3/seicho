(function(root){
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
function replacePostRoute(historyObject,path){historyObject.replaceState(null,'',postRoute(path))}
function clearPostRoute(historyObject){historyObject.replaceState(null,'','/')}
const api={postRoute,postPathFromRoute,replacePostRoute,clearPostRoute};
if(typeof module==='object'&&module.exports)module.exports=api;
else Object.assign(root,api);
})(typeof globalThis==='object'?globalThis:this);
