import NavBar from "../nav/NavBar";
import SiteFooter from "./SiteFooter";

export default function PlaygroundLayout({
    children,
}: {
    children: React.ReactNode;
}) {
    return (
        <div
            style={{
                width: "100vw",
                height: "100vh",
                overflow: "hidden",
                display: "flex",
                flexDirection: "column",
            }}
        >
            <NavBar />
            <div style={{ flex: 1, minHeight: 0, overflow: "hidden" }}>
                {children}
            </div>
            <SiteFooter />
        </div>
    );
}
