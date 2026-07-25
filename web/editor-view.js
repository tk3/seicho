(function(root){
function populatePostFields(document,fields){
 fields.path.value=document.path;
 fields.frontMatter.value=document.frontMatter;
 fields.body.value=document.body;
}
function setEditorVisible(editor,empty,visible){
 editor.classList.toggle('hidden',!visible);
 empty.classList.toggle('hidden',visible);
}
function setDialogVisible(dialog,visible){dialog.classList.toggle('hidden',!visible)}
function setControlsDisabled(controls,disabled){controls.forEach(control=>{control.disabled=disabled})}
function setDisclosureState(button,panel,expanded){button.setAttribute('aria-expanded',String(expanded));panel.classList.toggle('hidden',!expanded)}
const api={populatePostFields,setEditorVisible,setDialogVisible,setControlsDisabled,setDisclosureState};
if(typeof module==='object'&&module.exports)module.exports=api;
else Object.assign(root,api);
})(typeof globalThis==='object'?globalThis:this);
