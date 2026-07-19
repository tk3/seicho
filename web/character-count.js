(function(root){
function characterLength(value,locale='en'){
 const text=String(value).replace(/[\r\n]/g,'');
 if(typeof Intl.Segmenter!=='function')return Array.from(text).length;
 let count=0;
 for(const unused of new Intl.Segmenter(locale,{granularity:'grapheme'}).segment(text))count++;
 return count;
}
function formatCharacterCount(count,locale='en'){
 const number=count.toLocaleString(locale==='ja'?'ja-JP':'en-US');
 return locale==='ja'?number+'文字':number+' '+(count===1?'character':'characters');
}
const api={characterLength,formatCharacterCount};
if(typeof module==='object'&&module.exports)module.exports=api;
else Object.assign(root,api);
})(typeof globalThis==='object'?globalThis:this);
