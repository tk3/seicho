const test=require('node:test');
const assert=require('node:assert/strict');
const {populatePostFields,setEditorVisible,setDialogVisible}=require('../web/editor-view.js');

function classList(){
 const classes=new Set(['hidden']);
 return {toggle(name,force){if(force)classes.add(name);else classes.delete(name)},contains:name=>classes.has(name)};
}

test('populates all post editor fields',()=>{
 const fields={path:{value:''},frontMatter:{value:''},body:{value:''}};
 populatePostFields({path:'posts/hello.md',frontMatter:'title: Hello',body:'\nBody'},fields);
 assert.deepEqual({path:fields.path.value,frontMatter:fields.frontMatter.value,body:fields.body.value},{path:'posts/hello.md',frontMatter:'title: Hello',body:'\nBody'});
});

test('switches between the editor and empty view',()=>{
 const editor={classList:classList()},empty={classList:classList()};
 setEditorVisible(editor,empty,true);
 assert.equal(editor.classList.contains('hidden'),false);
 assert.equal(empty.classList.contains('hidden'),true);
 setEditorVisible(editor,empty,false);
 assert.equal(editor.classList.contains('hidden'),true);
 assert.equal(empty.classList.contains('hidden'),false);
});

test('opens and closes dialogs',()=>{
 const dialog={classList:classList()};
 setDialogVisible(dialog,true);
 assert.equal(dialog.classList.contains('hidden'),false);
 setDialogVisible(dialog,false);
 assert.equal(dialog.classList.contains('hidden'),true);
});
