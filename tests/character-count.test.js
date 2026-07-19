const test=require('node:test');
const assert=require('node:assert/strict');
const {characterLength,formatCharacterCount}=require('../web/character-count.js');

test('counts Japanese and ASCII characters',()=>{
 assert.equal(characterLength('清張 Seicho','ja'),9);
});

test('does not count line breaks',()=>{
 assert.equal(characterLength('one\r\ntwo\nthree','en'),11);
});

test('counts a joined emoji as one visible character',()=>{
 assert.equal(characterLength('👨‍👩‍👧‍👦','ja'),1);
});

test('formats Japanese character counts',()=>{
 assert.equal(formatCharacterCount(1234,'ja'),'1,234文字');
});

test('formats singular and plural English character counts',()=>{
 assert.equal(formatCharacterCount(1,'en'),'1 character');
 assert.equal(formatCharacterCount(1234,'en'),'1,234 characters');
});
