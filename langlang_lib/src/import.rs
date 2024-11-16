use std::collections::HashMap;
use std::path::{Path, PathBuf};
use std::{fs, io};

use langlang_syntax::visitor::Visitor;
use langlang_syntax::{ast, parser};

#[derive(Debug)]
pub enum Error {
    NameError(String),
    FileNotFound(String),
    PermissionDenied(String),
    OtherIOError(String),
    InvalidArgument(String),
    ParsingError(String),
}

impl From<io::Error> for Error {
    fn from(e: io::Error) -> Self {
        match e.kind() {
            io::ErrorKind::NotFound => Self::FileNotFound(e.to_string()),
            io::ErrorKind::PermissionDenied => {
                Self::PermissionDenied("Permission denied".to_string())
            }
            _ => Self::OtherIOError(format!("IO Error: {}", e.kind())),
        }
    }
}

impl From<parser::Error> for Error {
    fn from(e: parser::Error) -> Self {
        Self::ParsingError(e.to_string())
    }
}

pub trait ImportLoader {
    fn get_path(&self, import_path: &Path, parent_path: &Path) -> Result<PathBuf, Error>;
    fn get_content(&self, path: &Path) -> Result<String, Error>;
}

pub struct ImportResolver<T: ImportLoader> {
    loader: T,
}

impl<T: ImportLoader> ImportResolver<T> {
    pub fn new(loader: T) -> Self {
        Self { loader }
    }

    pub fn resolve(&self, source: &Path) -> Result<ast::Grammar, Error> {
        let mut r = self.resolve_import(source, source)?;
        let builtins = parser::parse(include_str!("./builtins.peg"))?;
        for def in builtins.definitions.values() {
            r.grammar.add_definition(def);
        }
        Ok(r.grammar)
    }

    fn resolve_import<'a>(
        &'a self,
        import_path: &'a Path,
        parent_path: &'a Path,
    ) -> Result<ImporterResolverFrame, Error> {
        let mut frame = self.create_frame(import_path, parent_path)?;
        let imports = frame.grammar.imports.to_owned();

        for import_node in &imports {
            let import_node_path = Path::new(&import_node.path);
            let imported_frame = self.resolve_import(import_node_path, &frame.import_path)?;

            for name in &import_node.names {
                match imported_frame.grammar.definitions.get(name) {
                    None => {
                        return Err(Error::NameError(format!(
                            "{} does not provide {}",
                            import_node.path, name,
                        )))
                    }
                    Some(imported_def) => {
                        // Add the imported definition to the parent frame's grammar and
                        // find all definitions that the imported definition depend on
                        frame.grammar.add_definition(imported_def);
                        for dep in imported_frame.find_definition_deps(imported_def) {
                            frame.grammar.add_definition(dep);
                        }
                    }
                }
            }
        }

        frame.grammar.imports = vec![];

        Ok(frame)
    }

    fn create_frame<'a>(
        &'a self,
        import_path: &'a Path,
        parent_path: &'a Path,
    ) -> Result<ImporterResolverFrame, Error> {
        let import_path = self.loader.get_path(import_path, parent_path)?;
        let grammar_str = self.loader.get_content(&import_path)?;
        let grammar = parser::parse(&grammar_str)?;
        Ok(ImporterResolverFrame {
            import_path,
            grammar,
        })
    }
}

struct ImporterResolverFrame {
    import_path: PathBuf,
    grammar: ast::Grammar,
}

impl ImporterResolverFrame {
    fn find_definition_deps<'a>(&'a self, def: &'a ast::Definition) -> Vec<&'a ast::Definition> {
        let mut f = DepFinder::new(&self.grammar);
        f.visit_definition(def);
        f.deps.into_values().collect()
    }
}

struct DepFinder<'ast> {
    grammar: &'ast ast::Grammar,
    deps: HashMap<&'ast String, &'ast ast::Definition>,
}

impl<'ast> DepFinder<'ast> {
    fn new(grammar: &'ast ast::Grammar) -> Self {
        Self {
            grammar,
            deps: HashMap::new(),
        }
    }
}

impl<'ast> Visitor<'ast> for DepFinder<'ast> {
    fn visit_identifier(&mut self, n: &'ast ast::Identifier) {
        if self.deps.get(&n.name).is_none() {
            let def = &self.grammar.definitions[&n.name];
            self.deps.insert(&n.name, def);
            self.visit_definition(def);
        }
    }

    fn visit_label(&mut self, n: &'ast ast::Label) {
        if self.deps.get(&n.label).is_none() {
            if let Some(def) = self.grammar.definitions.get(&n.label) {
                self.deps.insert(&n.label, def);
                self.visit_definition(def);
            }
        }
    }
}

#[derive(Default)]
pub struct RelativeImportLoader;

impl ImportLoader for RelativeImportLoader {
    fn get_path(&self, import_path: &Path, parent_path: &Path) -> Result<PathBuf, Error> {
        if import_path == parent_path {
            // Root node handling
            return Ok(import_path.to_path_buf());
        }
        let base_path = match parent_path.parent() {
            Some(p) => p,
            None => {
                return Err(Error::FileNotFound(format!(
                    "cannot retrieve parent directory: {}",
                    parent_path.display(),
                )))
            }
        };
        match import_path.strip_prefix("./") {
            Ok(relative_path) => Ok(base_path.join(relative_path)),
            Err(_) => Err(Error::InvalidArgument(format!(
                "Path isn't relative to the import site (should start with './'): {}",
                import_path.display()
            ))),
        }
    }

    fn get_content(&self, path: &Path) -> Result<String, Error> {
        Ok(fs::read_to_string(path)?)
    }
}

#[derive(Default)]
pub struct InMemoryImportLoader<'a> {
    grammars: HashMap<&'a str, &'a str>,
}

impl<'a> InMemoryImportLoader<'a> {
    pub fn add_grammar(&mut self, name: &'a str, grammar: &'a str) {
        self.grammars.insert(name, grammar);
    }
}

impl<'a> ImportLoader for InMemoryImportLoader<'a> {
    fn get_path(&self, import_path: &Path, _: &Path) -> Result<PathBuf, Error> {
        Ok(import_path.to_path_buf())
    }

    fn get_content(&self, path: &Path) -> Result<String, Error> {
        let p = match path.to_str() {
            Some(p) => p,
            None => {
                return Err(Error::InvalidArgument(format!(
                    "Invalid path: {}",
                    path.display()
                )))
            }
        };
        match self.grammars.get(&p) {
            Some(grammar) => Ok(grammar.to_string()),
            None => Err(Error::FileNotFound(format!(
                "Grammar {} not registered",
                path.display(),
            ))),
        }
    }
}
