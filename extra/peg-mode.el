;;; peg-mode.el --- Syntax highlight for Parsing Expression Grammars
;;
;; Copyright (C) 2018-2019  Lincoln Clarete
;;
;; Author: Lincoln de Sousa <lincoln@clarete.li>
;; Version: 0.0.1
;; Homepage: https://github.com/clarete/emacs.d
;;
;; This program is free software; you can redistribute it and/or modify
;; it under the terms of the GNU General Public License as published by
;; the Free Software Foundation, either version 3 of the License, or
;; (at your option) any later version.
;;
;; This program is distributed in the hope that it will be useful,
;; but WITHOUT ANY WARRANTY; without even the implied warranty of
;; MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
;; GNU General Public License for more details.
;;
;; You should have received a copy of the GNU General Public License
;; along with this program.  If not, see <https://www.gnu.org/licenses/>.
;;
;;; Commentary:
;;
;; Didn't really find any mode for PEGs that I liked.  Here's what I
;; put together to edit files that follow the syntax that Bryan Ford
;; defined in the paper he introduced PEGs.
;;
;;; Code:

(defconst peg-mode-syntax-table
  (let ((table (make-syntax-table)))
    ;; [] should also look like strings
    (modify-syntax-entry ?\[ "|" table)
    (modify-syntax-entry ?\] "|" table)
    ;; ' is a string delimiter
    (modify-syntax-entry ?' "\"" table)
    ;; " is a string delimiter too
    (modify-syntax-entry ?\" "\"" table)
    ;; Comments start with #
    (modify-syntax-entry ?# "< b" table)
    ;; \n is a comment ender
    (modify-syntax-entry ?\n "> b" table)
    table))

(defvar peg-font-lock-defaults
  `((
     ;; Color the name of the rule
     ("^\s*\\([a-zA-Z_][a-zA-Z0-9_]*\\)\s*<-" 1 'font-lock-function-name-face)
     ;; Color for the little assignment arrow
     ("<-" . font-lock-type-face)
     ;; ! & * + ? ( ) / are operators
     ("!\\|&\\|*\\|+\\|?\\|(\\|)\\|/" . font-lock-builtin-face)
     ;; Color for label
     ("\\(\\^[a-zA-Z_][a-zA-Z0-9_]*\\)" 1 'font-lock-constant-face)
     ;; Color for assignment of a name to a piece of the expression.
     ("\\(:[^\s]+\\)" 1 'font-lock-variable-name-face))))

;;;###autoload
(define-derived-mode peg-mode prog-mode "PEG Mode"
  :syntax-table peg-mode-syntax-table
  (set (make-local-variable 'comment-start) "#")
  (set (make-local-variable 'comment-end) "")
  (set (make-local-variable 'font-lock-defaults)
       peg-font-lock-defaults)
  (font-lock-ensure))

;;;###autoload
(add-to-list 'auto-mode-alist '("\\.peg\\'" . peg-mode))
(add-to-list 'auto-mode-alist '("\\.pegx\\'" . peg-mode))

(provide 'peg-mode)
;;; peg-mode.el ends here
