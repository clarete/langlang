"use client";

import React from "react";
import ReactDOM from "react-dom/client";
import { createBrowserRouter, RouterProvider } from "react-router-dom";

import "./global.css";

import DocLayout from "./layout/DocLayout";
import PlaygroundLayout from "./layout/PlaygroundLayout";
import PlaygroundPage from "./pages/PlaygroundPage";
import HomePage from "./pages/HomePage";
import GettingStarted from "./docs/getting-started.mdx";
import LanguageReference from "./docs/language-reference.mdx";
import Examples from "./docs/examples.mdx";

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
            return this.props.fallback ?? <div>Error loading page</div>;
        }
        return this.props.children;
    }
}

const router = createBrowserRouter([
    {
        path: "/",
        element: <DocLayout><HomePage /></DocLayout>,
    },
    {
        path: "/getting-started",
        element: <DocLayout><GettingStarted /></DocLayout>,
    },
    {
        path: "/language-reference",
        element: <DocLayout><LanguageReference /></DocLayout>,
    },
    {
        path: "/examples",
        element: <DocLayout><Examples /></DocLayout>,
    },
    {
        path: "/playground",
        element: <PlaygroundLayout><PlaygroundPage /></PlaygroundLayout>,
    },
], { basename: import.meta.env.BASE_URL.replace(/\/$/, "") || "/" });

ReactDOM.hydrateRoot(
    document.getElementById("app") as HTMLElement,
    <React.StrictMode>
        <ErrorBoundary>
            <RouterProvider router={router} />
        </ErrorBoundary>
    </React.StrictMode>,
);
