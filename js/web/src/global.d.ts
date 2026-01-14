/// <reference types="vite/client" />

export {};

declare global {
    interface FileSystemFileHandle {
        kind: "file";
        name: string;
        getFile(): Promise<File>;
        createWritable(): Promise<FileSystemWritableFileStream>;
    }

    interface Window {
        showOpenFilePicker(options?: {
            multiple?: boolean;
            excludeAcceptAllOption?: boolean;
            types?: {
                description?: string;
                accept: Record<string, string[]>;
            }[];
        }): Promise<FileSystemFileHandle[]>;

        FileSystemObserver: {
            new (
                callback: (records: FileSystemObserverRecord[]) => void,
            ): FileSystemObserver;
        };
    }

    interface FileSystemObserverRecord {
        type: "modified" | "appeared" | "disappeared" | "moved" | "unknown";
        readonly changedHandle: FileSystemFileHandle;
        readonly root: FileSystemFileHandle;
        handle: FileSystemHandle;
    }

    interface FileSystemObserver {
        observe(
            handle: FileSystemHandle,
            options?: { recursive?: boolean },
        ): Promise<void>;
        unobserve(handle: FileSystemFileHandle): void;
        disconnect(): void;
    }
}
