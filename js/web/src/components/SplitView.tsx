import { useCallback, useEffect, useRef, useState } from "react";
import {
    SplitViewContainer,
    SplitViewHandle,
    SplitViewHandleKnob,
    SplitViewRoot,
} from "./SplitView.styles";

interface HorizontalSplitViewProps {
    left: React.ReactNode;
    right: React.ReactNode;
    initialRatio?: number;
}

interface VerticalSplitViewProps {
    top: React.ReactNode;
    bottom: React.ReactNode;
    initialRatio?: number;
}

function SplitView(props: HorizontalSplitViewProps | VerticalSplitViewProps) {
    const [ratio, setRatio] = useState(() => {
        const v = props.initialRatio;
        if (typeof v !== "number") return 0.5;
        if (Number.isNaN(v)) return 0.5;
        return Math.min(0.9, Math.max(0.1, v));
    });

    const rootRef = useRef<HTMLDivElement>(null);

    const [isDragging, setIsDragging] = useState(false);

    const left = "top" in props ? props.top : props.left;
    const right = "top" in props ? props.bottom : props.right;

    const isHorizontal = "top" in props;

    const handleMouseMove = useCallback(
        (e: MouseEvent) => {
            e.preventDefault();
            e.stopPropagation();
            if (isDragging) {
                const root = rootRef.current;
                if (!root) return;
                const rect = root.getBoundingClientRect();
                const client = isHorizontal ? e.clientY : e.clientX;
                const start = isHorizontal ? rect.top : rect.left;
                const size = isHorizontal ? rect.height : rect.width;
                if (size <= 0) return;
                const next = (client - start) / size;
                setRatio(Math.min(0.9, Math.max(0.1, next)));
            }
        },
        [isDragging, isHorizontal],
    );

    const handleMouseUp = useCallback(() => {
        setIsDragging(false);
    }, []);

    const handleMouseDown = (e: React.MouseEvent<HTMLDivElement>) => {
        e.preventDefault();
        e.stopPropagation();
        setIsDragging(true);
    };

    useEffect(() => {
        window.addEventListener("mousemove", handleMouseMove);
        window.addEventListener("mouseup", handleMouseUp);
        return () => {
            window.removeEventListener("mousemove", handleMouseMove);
            window.removeEventListener("mouseup", handleMouseUp);
        };
    }, [handleMouseMove, handleMouseUp]);

    return (
        <SplitViewRoot ref={rootRef} horizontal={isHorizontal}>
            <SplitViewContainer horizontal={isHorizontal} span={ratio}>
                {left}
            </SplitViewContainer>
            <SplitViewHandle
                span={ratio}
                onMouseDown={handleMouseDown}
                onMouseUp={handleMouseUp}
                horizontal={isHorizontal}
            >
                <SplitViewHandleKnob horizontal={isHorizontal} />
            </SplitViewHandle>
            <SplitViewContainer horizontal={isHorizontal} span={1 - ratio}>
                {right}
            </SplitViewContainer>
        </SplitViewRoot>
    );
}

export default SplitView;
