---
type: "always_apply"
---

When writing any code, adhere to the following rules:
- Always read the files and get the context before making any edit changes in any of the files.
- All functions should follow Single Responsibility Principle.
- Add documentation for each function which describes the contract of that function.
- Inject dependencies wherever possible to allow writing UTs with fake dependencies.
- Follow existing patterns and standards in similar or adjacent code in the codebase.
- Use existing methods and implementations wherever possible. If this doesn't exist, use the language standard library or existing third party libraries. Avoid writing custom implementations wherever possible.
- Avoid code duplication as much as possible.

# GoLang guidelines
When writing code in GoLang, adhere to the following rules:
- Always make code changes before adding import statements. IDE might remove unused import statements if you add import statements first.

When writing test in GoLang, adhere to the following additional rules:
- All test function names should follow camel case.
- Prefer using fakes over mocks for dependencies.

If there is uncertainty, refer back the effective go documentation from Golang directly i.e. https://go.dev/doc/effective_go