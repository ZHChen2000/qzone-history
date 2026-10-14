'use strict';
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const template = fs.readFileSync(path.join(__dirname, '../pkg/export/templates/viewer.html'), 'utf8');
function load(payload) {
  const html = template.replaceAll('{{.PayloadB64}}', Buffer.from(JSON.stringify(payload)).toString('base64')).replaceAll('{{.UserQQ}}', payload.userQQ).replaceAll('{{.GeneratedAt}}', payload.generatedAt || '2026-10-06 02:29:07');
  const nodes = new Map();
  function element(id) {
    if (!nodes.has(id)) nodes.set(id, { value: id === 'time-order' ? 'desc' : id === 'duplicate-mode' ? 'fold' : 'all', innerHTML: '', style: {}, classList: {add(){},remove(){}}, addEventListener(){} });
    return nodes.get(id);
  }
  const context = vm.createContext({ TextDecoder, Uint8Array, atob, document: { getElementById: element, querySelectorAll: () => [] } });
  new vm.Script(html.match(/<script>([\s\S]*?)<\/script>/)[1]).runInContext(context, {timeout:15000});
  return { context, element, html, run: code => vm.runInContext(code, context) };
}
const original = {id:'original', userQQ:'self', senderQQ:'self', content:'山顶的月亮', timestamp:'2020-09-30T23:59:19+08:00', isReconstructed:false};
const like = {id:'like', userQQ:'self', senderQQ:'friend1', content:'\\t用户甲 ：山顶的月亮\\t', timeText:'2020年10月2日 21:30', isReconstructed:true, imageURLs:['https://example.invalid/one.jpg']};
const comment = {id:'comment', userQQ:'self', senderQQ:'friend2', content:'\\t\\t用户甲 ：山顶的月亮\\t\\t', timeText:'2025年1月25日 11:05', isReconstructed:true, comments:[{content:'评论内容',userQQ:'friend2'}]};
const other = {id:'other', userQQ:'self', senderQQ:'friend3', content:'好友乙：山顶的月亮', timeText:'2021年1月1日 10:00', isReconstructed:true};
const forwarded = {id:'forward', userQQ:'self', senderQQ:'friend4', content:'用户甲 ：山顶的月亮', timeText:'2022年1月1日 10:00', isReconstructed:true};
const payload = {userQQ:'self', moments:[original,like,comment,other,forwarded], boardMessages:[], activities:[
{receiverQQ:'self', content:like.content,type:2,senderQQ:'friend1'},
{receiverQQ:'self', content:comment.content,type:3,senderQQ:'friend2'},
{receiverQQ:'self', content:other.content,type:3,senderQQ:'friend3'},
{receiverQQ:'self', content:forwarded.content,type:1,senderQQ:'friend4'}]};
const page=load(payload);
const groups=page.run('groupMoments(DATA.moments,true)');
assert.equal(groups.length,3,'original + two self interactions fold; other source and forward stay separate');
const combined=groups.find(group=>group.records.length===3);
assert(combined);assert.equal(combined.representative.id,'original');
assert.equal(groups.reduce((n,g)=>n+g.records.length,0),5);
const markup=page.element('panel-moments').innerHTML;
assert.equal((markup.match(/class="record-detail"/g)||[]).length,5);
assert(markup.includes('含转发线索'));assert(markup.includes('原文前缀：好友乙'));assert(markup.includes('one.jpg'));assert(markup.includes('评论内容'));
assert.equal(page.run('JSON.stringify(DATA)'),JSON.stringify(payload),'rendering must not mutate records');
for(const mode of ['asc','desc']) {
page.element('time-order').value=mode;page.run('renderAll()');
const keys=page.run('groupMoments(DATA.moments,true).map(g=>sortKey(g.representative))');
assert(keys.every((v,i)=>i===0||(mode==='asc'?v>=keys[i-1]:v<=keys[i-1])));
}
page.element('duplicate-mode').value='all';page.run('renderMoments()');assert.equal((page.element('panel-moments').innerHTML.match(/class="card moment-group"/g)||[]).length,5);
page.element('moment-filter').value='visible';page.run('renderMoments()');assert.equal((page.element('panel-moments').innerHTML.match(/class="record-detail"/g)||[]).length,1);
page.run("searchQuery='__missing_text__';renderMoments()");assert(page.element('panel-moments').innerHTML.includes('没有匹配的说说'));
const ambiguous=load({...payload,moments:[original,{...original,id:'second-original'},like,comment]});
assert.equal(ambiguous.run('groupMoments(DATA.moments,true).length'),3,'multiple real posts must not be conflated with a reconstructed group');
const uncertain=load({...payload,moments:[original,other]});
assert.equal(uncertain.run('groupMoments(DATA.moments,true).length'),2,'unknown author prefix must not attach to original');
const empty=load({userQQ:'self',moments:[{id:'a',isReconstructed:true,content:''},{id:'b',isReconstructed:true,content:''}],activities:[],boardMessages:[]});
assert.equal(empty.run('groupMoments(DATA.moments,true).length'),2,'empty text is not evidence of duplication');
const dateCases=[
[{timestamp:'0001-01-01T00:00:00Z',timeText:'\\t2019年7月19日 19:47\\t'},Date.parse('2019-07-19T19:47:00+08:00')],
[{timeText:'tttt2021年12月31日 19:45tttt'},Date.parse('2021-12-31T19:45:00+08:00')],
[{timeText:'2020年2月29日 09:25'},Date.parse('2020-02-29T09:25:00+08:00')],
[{timeText:'2021年2月29日 09:25'},null],
[{timeText:'9月6日 02:59'},null],
[{timeText:'昨天 19:37'},null],
[{timeText:'2020年2月29日 25:25'},null]
];
for(const [item,expected] of dateCases){page.context.item=item;assert.equal(page.run('sortKey(item)'),expected);}
page.context.items=[{timeText:'9月6日 02:59'},{timeText:'2020年2月29日 09:25'},{timeText:'2019年2月28日 09:25'},{timeText:''}];
for(const mode of ['asc','desc']) {page.element('time-order').value=mode;const keys=page.run('items.slice().sort(compareByTime).map(sortKey)');assert.equal(keys[2],null);assert.equal(keys[3],null);}
if(process.argv[2]) {
const realHTML=fs.readFileSync(process.argv[2],'utf8');const encoded=JSON.parse(realHTML.match(/const DATA = JSON\.parse\(decodePayload\(("[^"\r\n]*")\)\);/)[1]);
const realData=JSON.parse(Buffer.from(encoded,'base64').toString('utf8'));const real=load(realData);
const realGroups=real.run('groupMoments(DATA.moments,true)');
assert.equal(realGroups.reduce((n,g)=>n+g.records.length,0),realData.moments.length);
assert.equal((real.element('panel-moments').innerHTML.match(/class="record-detail"/g)||[]).length,realData.moments.length);
const fragment=process.argv[3];
const sample=fragment ? realGroups.find(g=>g.records.some(m=>(m.content||'').includes(fragment))) : null;
if(fragment){assert(sample);assert.equal(sample.records.length,3);assert.equal(sample.representative.isReconstructed,false);}
console.log(JSON.stringify({privateLocalCheck:true,records:realData.moments.length,groups:realGroups.length,foldedGroups:realGroups.filter(g=>g.records.length>1).length,reportedExampleRecords:sample ? sample.records.length : null}));
}
console.log('Viewer regression checks passed: dates, source separation, originals, fold/full mode, search, metadata, and immutable data.');
