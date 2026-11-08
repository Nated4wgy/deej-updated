const assert=require('node:assert/strict');
const fs=require('node:fs/promises');
const path=require('node:path');
const {chromium}=require(process.env.PLAYWRIGHT_CORE_PATH||'playwright-core');
(async()=>{
 const root=path.resolve(__dirname,'..');const web=path.join(root,'pkg/deej/settingsui/web');
 const html=(await fs.readFile(path.join(web,'index.html'),'utf8')).replace('/*STYLE*/',await fs.readFile(path.join(web,'styles.css'),'utf8')).replace('/*SCRIPT*/',await fs.readFile(path.join(web,'app.js'),'utf8'));
 const browser=await chromium.launch({channel:'msedge',headless:true,chromiumSandbox:true});
 try{
  const page=await browser.newPage({viewport:{width:1180,height:850}});const errors=[];page.on('pageerror',error=>errors.push(error.message));
  await page.goto('about:blank');
  await page.evaluate(()=>{
   window.fixture={settings:{slider_mapping:{0:['master'],1:['discord.exe']},com_port:'COM4',baud_rate:9600,invert_sliders:false,noise_reduction:'default'},revision:'initial',startup:false,ports:['COM4','COM7'],filename:'C:/Deej/config.yaml',version:'test-preview',has_backup:true,live:true,saved:null,conflict:false};
   window.chrome={webview:{postMessage:message=>{const req=JSON.parse(message);const f=window.fixture;let result=null,error='';switch(req.method){
    case 'initial':result={settings:structuredClone(f.settings),revision:f.revision,startup:f.startup,ports:f.ports,filename:f.filename,version:f.version,has_backup:f.has_backup};break;
    case 'status':result={live:f.live,status:{serial:{connected:true,port:'COM4',positions:[.3,.5,.8,0,1],last_frame:new Date().toISOString()},audio:{targets:['discord.exe','firefox.exe','master'],p95_apply_ms:2.4,config_revision:f.revision}}};break;
    case 'save':if(f.conflict){error='settings changed outside this window';break;}f.saved=structuredClone(req.params);f.settings=structuredClone(req.params.settings);f.startup=req.params.startup;f.revision='saved';result=f.revision;break;
    case 'ports':result=f.ports;break;
    case 'restore':f.settings.slider_mapping={0:['master']};f.revision='restored';break;
   }queueMicrotask(()=>window.deejReceive({id:req.id,result,error}));}}};
  });
  await page.setContent(html);await page.getByRole('heading',{name:'Everything in balance.'}).waitFor();await page.getByText('Connected · COM4',{exact:true}).waitFor();
  await page.waitForFunction(()=>document.querySelector('.meter').getAttribute('aria-valuenow')==='30');
  await page.getByRole('button',{name:'Add target to slider 1',exact:true}).click();await page.getByRole('button',{name:/Active application/}).click();
  await page.getByRole('button',{name:'Save changes',exact:true}).click();await page.getByText('Settings applied',{exact:true}).waitFor();
  const saved=await page.evaluate(()=>window.fixture.saved);assert.deepEqual(saved.settings.slider_mapping['0'],['master','deej.current']);assert.equal(saved.expected_revision,'initial');
  await page.getByRole('button',{name:'Dark mode',exact:true}).click();assert.equal(await page.locator('html').getAttribute('data-theme'),'dark');
  await page.getByRole('button',{name:'Device & preferences',exact:true}).click();await page.getByRole('button',{name:'Low',exact:true}).click();await page.getByRole('button',{name:'Save changes',exact:true}).click();await page.getByText('Settings applied',{exact:true}).waitFor();assert.equal((await page.evaluate(()=>window.fixture.saved)).settings.noise_reduction,'low');
  await page.getByRole('button',{name:'Mixer 5',exact:true}).click();await page.getByRole('button',{name:'Add target to slider 2',exact:true}).click();const hostile='<img src=x onerror=alert(1)>';await page.getByRole('textbox',{name:'Search targets',exact:true}).fill(hostile);await page.locator('#add-manual').click();assert.equal(await page.locator('.target-chips img').count(),0);assert.ok((await page.locator('.target-chips').allTextContents()).some(text=>text.includes(hostile)));
  await page.evaluate(()=>window.fixture.conflict=true);await page.getByRole('button',{name:'Save changes',exact:true}).click();await page.getByRole('alert').filter({hasText:'settings changed outside this window'}).waitFor();assert.equal(await page.getByRole('button',{name:'Save changes',exact:true}).isEnabled(),true);
  // Revert discards a pending edit and preserves the saved configuration.
  await page.getByRole('button',{name:'Revert',exact:true}).click();await page.getByRole('button',{name:'Discard changes',exact:true}).click();await page.waitForFunction(()=>document.querySelector('#save').disabled);assert.equal(await page.locator('.target-chips').filter({hasText:hostile}).count(),0);
  for(const viewport of [{width:1180,height:850},{width:900,height:680}]){await page.setViewportSize(viewport);assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),true);const footer=await page.locator('footer').boundingBox();assert.ok(footer.y+footer.height<=viewport.height+1,'save bar must remain visible');}
  assert.deepEqual(errors,[]);
  if(process.env.UI_SCREENSHOT_DIR){await fs.mkdir(process.env.UI_SCREENSHOT_DIR,{recursive:true});await page.setViewportSize({width:1180,height:850});await page.evaluate(()=>{window.fixture.live=false;window.fixture.conflict=false;});await page.waitForFunction(()=>document.querySelector('#engine-state').textContent==='Mixer offline');await page.waitForFunction(()=>Array.from(document.querySelectorAll('.meter-fill')).every(el=>parseFloat(getComputedStyle(el).width)<0.1));await page.screenshot({path:path.join(process.env.UI_SCREENSHOT_DIR,'deej-modern-dark.png')});await page.getByRole('button',{name:'Light mode',exact:true}).click();await page.screenshot({path:path.join(process.env.UI_SCREENSHOT_DIR,'deej-modern-light.png')});}
  console.log('UI checks passed: live positions, grouped targets, save acknowledgement, themes, device settings, safe text rendering, conflicts, revert and responsive layout.');
 }finally{await browser.close();}
})().catch(error=>{console.error(error);process.exitCode=1;});
