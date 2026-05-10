import { Link } from "react-router-dom";
import {
    DocFooter,
    DocFooterNav,
    DocFooterNavLink,
    DocFooterCredit,
} from "./DocLayout.styles";

const NAV_LINKS = [
    { label: "Home", to: "/" },
    { label: "Getting Started", to: "/getting-started" },
    { label: "Reference", to: "/language-reference" },
    { label: "Examples", to: "/examples" },
    { label: "Playground", to: "/playground" },
];

export default function SiteFooter() {
    return (
        <DocFooter>
            <DocFooterNav>
                {NAV_LINKS.map(({ label, to }) => (
                    <DocFooterNavLink key={to} as={Link as any} to={to}>
                        {label}
                    </DocFooterNavLink>
                ))}
                <DocFooterNavLink
                    href="https://github.com/clarete/langlang"
                    target="_blank"
                    rel="noopener noreferrer"
                >
                    GitHub
                </DocFooterNavLink>
            </DocFooterNav>
            <DocFooterCredit>
                Made with ♥ by{" "}
                <a
                    href="https://clarete.li"
                    target="_blank"
                    rel="noopener noreferrer"
                >
                    @clarete
                </a>
            </DocFooterCredit>
        </DocFooter>
    );
}
