import { chromium } from 'playwright'
import { readFileSync, writeFileSync } from 'node:fs'
import assert from 'node:assert/strict'
const session=JSON.parse(readFileSync('test-results/trial30-session.json','utf8'))
const browser=await chromium.launch({headless:true,channel:'msedge'})
const context=await browser.newContext({viewport:{width:1920,height:1080},locale:'zh-CN'})
await context.addInitScript(token=>{localStorage.setItem('token',token);localStorage.setItem('endpointURL',`${location.origin}/graphql`)},session.token)
const page=await context.newPage();const errors=[]
page.on('pageerror',error=>errors.push(error.message))
page.on('response',async response=>{if(response.url().endsWith('/graphql'))try{const data=await response.json();if(data.errors)errors.push(...data.errors.map(e=>e.message))}catch{}})
try{
 await page.goto('http://127.0.0.1:12074/')
 await page.getByRole('heading',{name:'运行状态',exact:true}).waitFor()
 await page.waitForTimeout(2000)
 assert(await page.locator('.runtime-heading').getByText('运行中',{exact:true}).count())
 await page.waitForFunction(()=>[...document.querySelectorAll('.runtime-stats strong')].some(e=>/KB|MB/.test(e.textContent)))
 await page.screenshot({path:'test-results/trial30-installed.png',fullPage:true})
 for(const label of ['配置','DNS','路由','群组','节点','订阅','总览']){
  await page.locator('.console-sidebar').getByRole('button',{name:new RegExp(`^${label}`)}).click()
  await page.getByRole('heading',{name:label,exact:true}).first().waitFor()
  await page.waitForTimeout(200)
 }
 await page.setViewportSize({width:390,height:844})
 await page.waitForTimeout(300)
 assert(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth))
 await page.screenshot({path:'test-results/trial30-mobile.png',fullPage:true})
 assert.equal(errors.length,0,JSON.stringify(errors))
 writeFileSync('test-results/trial30-browser-report.json',JSON.stringify({passed:true,errors,desktop:true,mobile:true},null,2))
 console.log('30.3 real UI, all management pages, desktop and mobile: passed')
}finally{await browser.close()}
