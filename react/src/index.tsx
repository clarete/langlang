import { test as wasmTest } from '@langlang/wasm';
import { useState, useEffect } from 'react';

export function useWasmTest() {
  const [message, setMessage] = useState<string>("");

  useEffect(() => {
    // Simulating async if needed, though test() is sync currently
    setMessage(wasmTest());
  }, []);

  return message;
}

export function TestComponent() {
  const message = useWasmTest();
  return <span>{message} (via @langlang/react)</span>;
}

