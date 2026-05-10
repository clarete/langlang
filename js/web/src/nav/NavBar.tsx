import { Link } from "react-router-dom";
import { NavRoot, NavContainer, NavLogo, NavLinks, NavLink } from "./NavBar.styles";
import ThemeToggle from "./ThemeToggle";

export default function NavBar() {
    return (
        <NavRoot>
            <NavContainer>
                <NavLogo as={Link as any} to="/">
                    {"{"} langlang {"}"}
                </NavLogo>
                <NavLinks>
                    <NavLink as={Link as any} to="/getting-started">
                        Getting Started
                    </NavLink>
                    <NavLink as={Link as any} to="/language-reference">
                        Reference
                    </NavLink>
                    <NavLink as={Link as any} to="/examples">
                        Examples
                    </NavLink>
                    <NavLink as={Link as any} to="/playground">
                        Playground
                    </NavLink>
                    <NavLink
                        href="https://github.com/clarete/langlang"
                        target="_blank"
                        rel="noopener noreferrer"
                    >
                        GitHub
                    </NavLink>
                </NavLinks>
                <ThemeToggle />
            </NavContainer>
        </NavRoot>
    );
}
