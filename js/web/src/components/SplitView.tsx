import { createRef, useCallback, useEffect, useState } from "react";
import {
	SplitViewContainer,
	SplitViewHandle,
	SplitViewHandleKnob,
	SplitViewRoot,
} from "./SplitView.styles";

interface HorizontalSplitViewProps {
	left: React.ReactNode;
	right: React.ReactNode;
}

interface VerticalSplitViewProps {
	top: React.ReactNode;
	bottom: React.ReactNode;
}

function SplitView(props: HorizontalSplitViewProps | VerticalSplitViewProps) {
	const [ratio, setRatio] = useState(0.5);

	const handleRef = createRef<HTMLDivElement>();

	const [isDragging, setIsDragging] = useState(false);

	const left = "top" in props ? props.top : props.left;
	const right = "top" in props ? props.bottom : props.right;

	const isHorizontal = "top" in props;

	const handleMouseMove = useCallback(
		(e: MouseEvent) => {
			e.preventDefault();
			e.stopPropagation();
			if (isDragging) {
				const client = isHorizontal ? e.clientY : e.clientX;
				const inner = isHorizontal ? window.innerHeight : window.innerWidth;

				setRatio(client / inner);
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
		<SplitViewRoot horizontal={isHorizontal}>
			<SplitViewContainer horizontal={isHorizontal} span={ratio}>{left}</SplitViewContainer>
			<SplitViewHandle
				ref={handleRef}
				span={ratio}
				onMouseDown={handleMouseDown}
				onMouseUp={handleMouseUp}
				horizontal={isHorizontal}
			>
				<SplitViewHandleKnob horizontal={isHorizontal} />
			</SplitViewHandle>
			<SplitViewContainer horizontal={isHorizontal} span={1 - ratio}>{right}</SplitViewContainer>
		</SplitViewRoot>
	);
}

export default SplitView;
