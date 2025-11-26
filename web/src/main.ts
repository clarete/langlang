import { test } from '@langlang/wasm';

document.querySelector<HTMLDivElement>('#app')!.innerHTML = `
  <div>
    <h1>Test App</h1>
    <p>${test()}</p>
  </div>
`

