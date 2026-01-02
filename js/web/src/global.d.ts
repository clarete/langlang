export {};

declare global {
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
        handle: FileSystemHandle;
    }

    interface FileSystemObserver {
        observe(
            handle: FileSystemHandle,
            options?: { recursive?: boolean },
        ): Promise<void>;
        disconnect(): void;
    }
}
