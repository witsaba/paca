//! Categories management UI
//!
//! TODO: Implement CRUD operations with Leptos reactivity

use leptos::prelude::*;

#[component]
pub fn CategoriesManager() -> impl IntoView {
    view! {
        <div class="categories-manager">
            <header>
                <h2>"Categories"</h2>
                <button class="btn-primary">"New Category"</button>
            </header>
            <table class="categories-table">
                <thead>
                    <tr>
                        <th>"Name"</th>
                        <th>"Actions"</th>
                    </tr>
                </thead>
                <tbody>
                    <tr>
                        <td colspan="2">"Loading categories..."</td>
                    </tr>
                </tbody>
            </table>
        </div>
    }
}
