import assert from 'node:assert/strict';
import {test} from 'node:test';
import JSZip from 'jszip';
import {projectImport,validateNativeImport} from './project-import';
import {createHash,webcrypto} from 'node:crypto';
async function archive(name='project.epru'){const zip=new JSZip();zip.file(name,'native fixture');return await zip.generateAsync({type:'nodebuffer',compression:'DEFLATE'});}
test('native import validates CRC, paths and root native data before any host call',async()=>{
 const data=await archive();assert.deepEqual(await validateNativeImport(data.toString('base64')),new Uint8Array(data));
 for(const name of ['x.txt','../project.epru','/project.epru','a\\project.epru'])await assert.rejects(validateNativeImport((await archive(name)).toString('base64')));
 await assert.rejects(validateNativeImport('garbage'));
 const bad=Buffer.from(data);const at=bad.indexOf(Buffer.from([0x50,0x4b,0x01,0x02]));bad[at+16]^=1;await assert.rejects(validateNativeImport(bad.toString('base64')));
});
for(const scenario of ['success','no-confirm','source-mismatch','existing-name','undefined','same-source','wrong-owner','hash-mismatch'])test('native import '+scenario,async()=>{
 const global=globalThis as unknown as {eda:unknown};const old=global.eda;let calls=0;
 if(!globalThis.crypto)Object.defineProperty(globalThis,'crypto',{value:webcrypto});
 const data=await archive();const name='restore-'+scenario;
 global.eda={dmt_Project:{getCurrentProjectInfo:async()=>({uuid:scenario==='source-mismatch'?'other':'source'}),getAllProjectsUuid:async()=>['existing'],getProjectInfo:async(id:string)=>({uuid:id,teamUuid:scenario==='wrong-owner'?'other':'team',friendlyName:id==='existing'?(scenario==='existing-name'?name:'old'):name})},sys_FileManager:{importProjectByProjectFile:async(file:File,type:string,props:unknown,target:Record<string,unknown>)=>{calls++;assert.equal(type,'EasyEDA Pro');assert.deepEqual(props,{importOption:'ImportDocument'});assert.equal(target.operation,'New Project');assert.equal(file.size,data.length);return scenario==='undefined'?undefined:{uuid:scenario==='same-source'?'source':'new'};}}};
 const payload={projectUuid:'source',teamUuid:'team',friendlyName:name,base64:data.toString('base64'),sha256:scenario==='hash-mismatch'?'0'.repeat(64):createHash('sha256').update(data).digest('hex'),allowDiscardUnsaved:scenario!=='no-confirm'};
 try{if(scenario==='success'){const a=await projectImport(payload);assert.equal(a.result?.restoreVerified,false);assert.equal(a.result?.identityVerified,true);assert.deepEqual(await projectImport(payload),a);assert.equal(calls,1);}else{await assert.rejects(projectImport(payload));if(['no-confirm','source-mismatch','existing-name','hash-mismatch'].includes(scenario))assert.equal(calls,0);else{await assert.rejects(projectImport(payload));assert.equal(calls,1);}}}finally{global.eda=old;}
});
