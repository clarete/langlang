import { TestComponent } from '@langlang/react'
import { useState } from 'react'
import './App.css'

function App() {
  const [count, setCount] = useState(0)

  return (
    <>
      <h1>@langlang/web-test</h1>
      <div className="card">
        <button onClick={() => setCount((count) => count + 1)} type="button">
          count is {count}
        </button>
        <p>
          <TestComponent />
        </p>
      </div>
    </>
  )
}

export default App
