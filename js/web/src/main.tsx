"use client";

import React, { Suspense } from "react";
import ReactDOM from "react-dom/client";
import App from "./App.tsx";

import "./global.css";

class ErrorBoundary extends React.Component<
    { children: React.ReactNode; fallback?: React.ReactNode },
    { hasError: boolean }
> {
    constructor(props: { children: React.ReactNode }) {
        super(props);
        this.state = { hasError: false };
    }

    static getDerivedStateFromError(_: Error) {
        return { hasError: true };
    }

    componentDidCatch(error: Error, info: React.ErrorInfo) {
        console.error(error, info);
    }

    render() {
        if (this.state.hasError) {
            return this.props.fallback ?? <div>Error</div>;
        }

        return this.props.children;
    }
}
ReactDOM.createRoot(document.getElementById("app") as HTMLElement).render(
    <React.StrictMode>
        <ErrorBoundary>
            <Suspense fallback={<div>Loading...</div>}>
                <App />
            </Suspense>
        </ErrorBoundary>
    </React.StrictMode>,
);
