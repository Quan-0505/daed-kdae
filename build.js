import fs from 'fs';
import path from 'path';
import { execSync } from 'child_process';

const distDir = path.join(process.cwd(), 'dist');
const zipFile = path.join(process.cwd(), 'daed-official', 'web.zip');

if (!fs.existsSync(distDir) || !fs.existsSync(path.join(distDir, 'index.html'))) {
  console.log('Extracting web.zip to dist/ ...');
  fs.mkdirSync(distDir, { recursive: true });
  execSync(`unzip -o -q "${zipFile}" -d _tmp_extract`);
  execSync(`cp -r _tmp_extract/web/* "${distDir}/"`);
  execSync(`rm -rf _tmp_extract`);
}

// Ensure index.html exists
const indexPath = path.join(distDir, 'index.html');
if (fs.existsSync(indexPath)) {
  let html = fs.readFileSync(indexPath, 'utf-8');
  if (!html.includes('daed-kdae')) {
    html = html.replace(
      '<title>daed</title>',
      `<base href="/" />\n    <title>daed-kdae</title>\n    <meta name="description" content="Web dashboard and transparent proxy controller for daed and kdae eBPF routing engine" />\n    <meta property="og:title" content="daed-kdae" />\n    <meta property="og:description" content="Web dashboard and transparent proxy controller for daed and kdae eBPF routing engine" />`
    );
    fs.writeFileSync(indexPath, html, 'utf-8');
  }
}

// Ensure Setup-CKyMGYKT.js renders username & password inputs cleanly
const setupFile = path.join(distDir, 'assets', 'Setup-CKyMGYKT.js');
if (fs.existsSync(setupFile)) {
  let setupJs = fs.readFileSync(setupFile, 'utf-8');

  // Fix 1: Ensure E(e) returns a valid number even if API returns unexpected format
  const s1_orig = "async function E(e){let{numberUsers:t}=await new _(f(e)).get(`/auth/status`);return t}";
  const s1_patched = "async function E(e){try{let res=await new _(f(e)).get(`/auth/status`);let t=res?.numberUsers;return typeof t==='number'?t:1}catch(err){return 1}}";
  if (setupJs.includes(s1_orig)) {
    setupJs = setupJs.replace(s1_orig, s1_patched);
  }

  // Fix 2: Ensure step 2 always renders the username and password form, whether s === 0 (create) or s > 0 (login), with toggle
  const s2_orig = "t===1&&s===0&&(0,S.jsx)(`form`,{onSubmit:H,children:(0,S.jsxs)(`div`,{className:`mx-auto w-full max-w-md space-y-4`,children:[(0,S.jsx)(g,{label:e(`username`),placeholder:`admin`,withAsterisk:!0,value:N.username,onChange:e=>P({...N,username:e.target.value}),error:F.username,icon:(0,S.jsx)(b,{className:`h-4 w-4`})}),(0,S.jsx)(g,{type:`password`,label:e(`password`),placeholder:`password`,withAsterisk:!0,value:N.password,onChange:e=>P({...N,password:e.target.value}),error:F.password,icon:(0,S.jsx)(y,{className:`h-4 w-4`})}),(0,S.jsx)(h,{type:`submit`,className:`w-full`,uppercase:!0,children:e(`actions.create account`)})]})}),t===1&&s>0&&(0,S.jsx)(`form`,{onSubmit:U,children:(0,S.jsxs)(`div`,{className:`mx-auto w-full max-w-md space-y-4`,children:[(0,S.jsx)(g,{label:e(`username`),placeholder:`admin`,withAsterisk:!0,value:L.username,onChange:e=>R({...L,username:e.target.value}),error:z.username,icon:(0,S.jsx)(b,{className:`h-4 w-4`})}),(0,S.jsx)(g,{type:`password`,label:e(`password`),placeholder:`password`,withAsterisk:!0,value:L.password,onChange:e=>R({...L,password:e.target.value}),error:z.password,icon:(0,S.jsx)(y,{className:`h-4 w-4`})}),(0,S.jsx)(h,{type:`submit`,className:`w-full`,uppercase:!0,children:e(`actions.login`)})]})})";
  const s2_patched = "t===1&&(0,S.jsxs)(`div`,{className:`mx-auto w-full max-w-md space-y-4`,children:[s===0?(0,S.jsx)(`form`,{onSubmit:H,children:(0,S.jsxs)(`div`,{className:`space-y-4`,children:[(0,S.jsx)(g,{label:e(`username`),placeholder:`admin`,withAsterisk:!0,value:N.username,onChange:e=>P({...N,username:e.target.value}),error:F.username,icon:(0,S.jsx)(b,{className:`h-4 w-4`})}),(0,S.jsx)(g,{type:`password`,label:e(`password`),placeholder:`password`,withAsterisk:!0,value:N.password,onChange:e=>P({...N,password:e.target.value}),error:F.password,icon:(0,S.jsx)(y,{className:`h-4 w-4`})}),(0,S.jsx)(h,{type:`submit`,className:`w-full`,uppercase:!0,children:e(`actions.create account`)})]})}):(0,S.jsx)(`form`,{onSubmit:U,children:(0,S.jsxs)(`div`,{className:`space-y-4`,children:[(0,S.jsx)(g,{label:e(`username`),placeholder:`admin`,withAsterisk:!0,value:L.username,onChange:e=>R({...L,username:e.target.value}),error:z.username,icon:(0,S.jsx)(b,{className:`h-4 w-4`})}),(0,S.jsx)(g,{type:`password`,label:e(`password`),placeholder:`password`,withAsterisk:!0,value:L.password,onChange:e=>R({...L,password:e.target.value}),error:z.password,icon:(0,S.jsx)(y,{className:`h-4 w-4`})}),(0,S.jsx)(h,{type:`submit`,className:`w-full`,uppercase:!0,children:e(`actions.login`)})]})}),(0,S.jsx)(`button`,{type:`button`,onClick:()=>c(s===0?1:0),className:`w-full text-center text-xs text-primary hover:underline cursor-pointer py-1`,children:s===0?`已有账号？点击登录`:`没有账号？点击创建管理员账号`})]})";
  if (setupJs.includes(s2_orig)) {
    setupJs = setupJs.replace(s2_orig, s2_patched);
  }

  fs.writeFileSync(setupFile, setupJs, 'utf-8');
}

console.log('Build completed successfully.');
