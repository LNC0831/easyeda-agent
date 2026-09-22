/// <reference types="@jlceda/pro-api-types" />
import JSZip from 'jszip';
import { ActionError, ErrorCodes, type ActionResult } from './protocol';
import { requireString } from './util';
const LIMIT = 16 * 1024 * 1024;
const EXPANDED = 128 * 1024 * 1024;
const crcTable = Array.from({length:256}, (_, n) => {for(let b=0;b<8;b++) n=n&1?0xedb88320^(n>>>1):n>>>1; return n>>>0;});
const attempts = new Map<string, Promise<ActionResult>>();
export async function validateNativeImport(encoded: string): Promise<Uint8Array> {
 if (!encoded || encoded.length > Math.ceil(LIMIT/3)*4 || encoded.length%4!==0 || !/^[A-Za-z0-9+/]*={0,2}$/.test(encoded)) throw new Error('Invalid or oversized native archive');
 const bytes=Uint8Array.from(atob(encoded), c=>c.charCodeAt(0));
 if (!bytes.length || bytes.length>LIMIT) throw new Error('Invalid archive size');
 const zip=await JSZip.loadAsync(bytes); const files=Object.values(zip.files);
 if(files.length>2048 || files.filter(f=>!f.dir && /^[^/\\]+\.epru$/i.test(f.name)).length!==1) throw new Error('Expected one root epru and at most 2048 entries');
 let total=0;
 for(const f of files){
  const entry=f as typeof f & {unsafeOriginalName?:string;_data?:{uncompressedSize?:number;crc32?:number}};
  if((entry.unsafeOriginalName && entry.unsafeOriginalName!==f.name) || f.name.startsWith('/') || /[\\:]/.test(f.name) || f.name.split('/').includes('..')) throw new Error('Unsafe archive path');
  if(f.dir) continue;
  const size=entry._data?.uncompressedSize, expectedCRC=entry._data?.crc32;
  if(!Number.isSafeInteger(size)||size!<0||size!>EXPANDED-total||!Number.isInteger(expectedCRC)) throw new Error('Invalid or oversized expanded archive');
  let length=0,crc=0xffffffff;
  await new Promise<void>((resolve,reject)=>{
   const source=f as typeof f & {internalStream(type:'uint8array'):JSZip.JSZipStreamHelper<Uint8Array>};
   if(typeof source.internalStream!=='function'){reject(new Error('Bounded ZIP streaming unavailable'));return;}
   const stream=source.internalStream('uint8array'); let failed=false;
   stream.on('data',(chunk:Uint8Array)=>{if(failed)return; length+=chunk.length; total+=chunk.length;
    if(length>size!||total>EXPANDED){failed=true;stream.pause();reject(new Error('Expanded archive limit exceeded'));return;}
    for(const b of chunk)crc=crcTable[(crc^b)&255]^(crc>>>8);
   });
   stream.on('error',reject); stream.on('end',()=>{if(failed)return; if(length!==size||((crc^0xffffffff)>>>0)!==(expectedCRC!>>>0))reject(new Error('Archive CRC/length mismatch'));else resolve();});stream.resume();
  });
 }
 return bytes;
}
export async function projectImport(payload:Record<string,unknown>):Promise<ActionResult>{
 const source=requireString(payload,'projectUuid'), team=requireString(payload,'teamUuid'), name=requireString(payload,'friendlyName'), encoded=requireString(payload,'base64'), digest=requireString(payload,'sha256');
 if(!source.trim()||!team.trim()||!name.trim()||!/^[a-f0-9]{64}$/.test(digest)) throw new Error('Explicit source project, owner, name and SHA-256 required');
 if(payload.allowDiscardUnsaved!==true) throw new ActionError(ErrorCodes.PRECONDITION_REFUSED,'Save all documents and acknowledge allowDiscardUnsaved:true before import');
 if(payload.existingProjectUuid!==undefined||payload.saveTo!==undefined) throw new Error('Only a new project target is supported');
 const bytes=await validateNativeImport(encoded);
 const actual=Array.from(new Uint8Array(await crypto.subtle.digest('SHA-256',new Uint8Array(bytes)))).map(b=>b.toString(16).padStart(2,'0')).join('');
 if(actual!==digest)throw new Error('Archive SHA-256 mismatch');
 const key=JSON.stringify([source,team,name,digest]);
 const previous=attempts.get(key);if(previous)return previous;
 if(attempts.size>=64)throw new Error('Import attempt budget exhausted; inspect previous outcomes');
 const task=(async():Promise<ActionResult>=>{
  const before=await eda.dmt_Project.getCurrentProjectInfo();if(before?.uuid!==source)throw new Error('Active source project mismatch');
  const existing=await eda.dmt_Project.getAllProjectsUuid(team);
  if(!Array.isArray(existing)||existing.some(id=>typeof id!=='string'))throw new Error('Cannot enumerate target team projects');
  for(const id of existing){const info=await eda.dmt_Project.getProjectInfo(id);if(!info)throw new Error('Cannot read existing target project');if(info.friendlyName===name)throw new Error('Target name already exists; inspect it instead of reimporting');}
  if((await eda.dmt_Project.getCurrentProjectInfo())?.uuid!==source)throw new Error('Source project changed during import preflight');
  let imported:IDMT_BriefProjectItem|undefined;
  try {imported=await eda.sys_FileManager.importProjectByProjectFile(new File([new Uint8Array(bytes)],'restore.epro2'), 'EasyEDA Pro', {importOption:'ImportDocument' as ESYS_ImportProjectImportOption}, {operation:'New Project',newProjectOwnerTeamUuid:team,newProjectFriendlyName:name});}
  catch(error){throw new Error('Import may have partially created a project; inspect before retrying: '+String(error));}
  if(!imported?.uuid || imported.uuid===source || existing.includes(imported.uuid))throw new Error('Import returned no distinct new project; partial side effects unknown, do not retry blindly');
  const info=await eda.dmt_Project.getProjectInfo(imported.uuid);
  if(info?.uuid!==imported.uuid||info.friendlyName!==name||info.teamUuid!==team)throw new Error('Imported identity/owner/name not verified; inspect project '+imported.uuid);
  return {result:{uuid:imported.uuid,friendlyName:info.friendlyName,teamUuid:info.teamUuid,sourceProjectUuid:source,sha256:digest,imported:true,identityVerified:true,restoreVerified:false,sourceUnchangedVerified:false}};
 })();attempts.set(key,task);return task;
}
