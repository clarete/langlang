import { CodeJar } from "codejar";
import * as sysdcl from "../pkg/sysdcl";

const node = document.querySelector('.editor');

const lang = sysdcl.Lang.new()

const my = editor => {
    let code = editor.textContent;
    if (editor.textContent.length > 0)
        code = lang.highlight(editor.textContent);
    editor.innerHTML = code;
};

const jar = CodeJar(node, my);
