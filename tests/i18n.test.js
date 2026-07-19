const test=require('node:test');
const assert=require('node:assert/strict');
const {messages,translate}=require('../web/i18n.js');

test('keeps Japanese and English translation keys aligned',()=>{
 assert.deepEqual(Object.keys(messages.ja).sort(),Object.keys(messages.en).sort());
});

test('returns localized messages and falls back to the key',()=>{
 assert.equal(translate('ja','save'),'保存');
 assert.equal(translate('en','save'),'Save');
 assert.equal(translate('en','missingKey'),'missingKey');
});
