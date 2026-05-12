import { useState } from "react";
import { Link } from "react-router-dom";
import {
    NavRoot,
    NavContainer,
    NavLogo,
    NavLinks,
    NavLink,
    HamburgerButton,
    MobileMenu,
    MobileNavLink,
} from "./NavBar.styles";
import ThemeToggle from "./ThemeToggle";

const NAV_LINKS = [
    { to: "/getting-started", label: "Getting Started" },
    { to: "/language-reference", label: "Reference" },
    { to: "/examples", label: "Examples" },
    { to: "/playground", label: "Playground" },
    { href: "https://github.com/clarete/langlang", label: "GitHub" },
] as const;

export default function NavBar() {
    const [open, setOpen] = useState(false);

    return (
        <NavRoot>
            <NavContainer>
                <NavLogo as={Link as any} to="/">
                    {"{"} langlang {"}"}
                </NavLogo>
                <NavLinks>
                    {NAV_LINKS.map((link) =>
                        "href" in link ? (
                            <NavLink
                                key={link.label}
                                href={link.href}
                                target="_blank"
                                rel="noopener noreferrer"
                            >
                                {link.label}
                            </NavLink>
                        ) : (
                            <NavLink key={link.label} as={Link as any} to={link.to}>
                                {link.label}
                            </NavLink>
                        )
                    )}
                </NavLinks>
                <ThemeToggle />
                <HamburgerButton
                    aria-label={open ? "Close menu" : "Open menu"}
                    aria-expanded={open}
                    onClick={() => setOpen((v) => !v)}
                >
                    {open ? "✕" : "☰"}
                </HamburgerButton>
            </NavContainer>
            {open && (
                <MobileMenu>
                    {NAV_LINKS.map((link) =>
                        "href" in link ? (
                            <MobileNavLink
                                key={link.label}
                                href={link.href}
                                target="_blank"
                                rel="noopener noreferrer"
                                onClick={() => setOpen(false)}
                            >
                                {link.label}
                            </MobileNavLink>
                        ) : (
                            <MobileNavLink
                                key={link.label}
                                as={Link as any}
                                to={link.to}
                                onClick={() => setOpen(false)}
                            >
                                {link.label}
                            </MobileNavLink>
                        )
                    )}
                </MobileMenu>
            )}
        </NavRoot>
    );
}
