//! Categories management UI with full CRUD operations

use leptos::{prelude::*, task::spawn_local};
use crate::components::api::{self, CreateCategory};

/// Category with path for breadcrumb display
#[derive(Debug, Clone)]
pub struct CategoryDisplay {
    pub id: i64,
    pub name: String,
    pub parent_id: Option<i64>,
    pub path: String,
}

/// Categories manager component with full CRUD
#[component]
pub fn CategoriesManager() -> impl IntoView {
    // State signals
    let categories = RwSignal::new(Vec::<CategoryDisplay>::new());
    let loading = RwSignal::new(false);
    let error = RwSignal::new(Option::<String>::None);
    
    // Form inputs
    let new_name = RwSignal::new(String::new());
    let new_parent_id = RwSignal::new(Option::<i64>::None);

    // Load categories
    let load_categories = {
        let categories = categories.clone();
        let loading = loading.clone();
        let error = error.clone();
        
        move || {
            loading.set(true);
            error.set(None);
            
            spawn_local(async move {
                match api::list_categories(None).await {
                    Ok(cats) => {
                        let display_cats: Vec<CategoryDisplay> = cats.into_iter().map(|c| {
                            CategoryDisplay {
                                id: c.category.id,
                                name: c.category.name,
                                parent_id: c.category.parent_id,
                                path: c.path,
                            }
                        }).collect();
                        categories.set(display_cats);
                        loading.set(false);
                    }
                    Err(e) => {
                        error.set(Some(e));
                        loading.set(false);
                    }
                }
            });
        }
    };

    // Initial load
    load_categories();

    // Create category handler
    let handle_create = {
        let new_name = new_name.clone();
        let new_parent_id = new_parent_id.clone();
        let error = error.clone();
        let load_categories = load_categories.clone();
        
        move || {
            let name = new_name.get().trim().to_string();
            if name.is_empty() {
                error.set(Some("Category name cannot be empty".into()));
                return;
            }
            let parent_id = new_parent_id.get();
            
            spawn_local(async move {
                match api::create_category(CreateCategory { name, parent_id }).await {
                    Ok(_) => {
                        new_name.set(String::new());
                        new_parent_id.set(None);
                        load_categories();
                    }
                    Err(e) => {
                        error.set(Some(e));
                    }
                }
            });
        }
    };

    // Delete category handler
    let handle_delete = {
        let error = error.clone();
        let load_categories = load_categories.clone();
        
        move |id: i64| {
            spawn_local(async move {
                match api::delete_category(id).await {
                    Ok(_) => {
                        load_categories();
                    }
                    Err(e) => {
                        error.set(Some(e));
                    }
                }
            });
        }
    };

    view! {
        <div class="categories-manager">
            <header>
                <h2>"Categories"</h2>
            </header>

            // Error display
            <Show when={move || error.get().is_some()}>
                <div class="error-message">
                    <span>{move || error.get().unwrap_or_default()}</span>
                    <button class="btn-close" on:click={move |_| error.set(None)}>"×"</button>
                </div>
            </Show>

            // Create new category
            <div class="create-category-form">
                <h3>"Add New Category"</h3>
                <div class="form-row">
                    <input
                        type="text"
                        placeholder="Category name"
                        prop:value={move || new_name.get()}
                        on:input={move |e| new_name.set(event_target_value(&e))}
                    />
                    <select
                        on:change={move |e| {
                            let val = event_target_value(&e);
                            new_parent_id.set(if val.is_empty() { None } else { val.parse().ok() });
                        }}
                    >
                        <option value="">"Root (no parent)"</option>
                        {move || categories.get().into_iter().map(|cat| {
                            view! { <option value={cat.id.to_string()}>{cat.path.clone()}</option> }
                        }).collect_view()}
                    </select>
                    <button class="btn-primary" on:click={move |_| handle_create()}>
                        "Add"
                    </button>
                </div>
            </div>

            // Categories list
            <div class="categories-list">
                <h3>"Existing Categories"</h3>
                
                <Show when={move || loading.get()}>
                    <p class="loading">"Loading categories..."</p>
                </Show>

                <Show when={move || !loading.get() && categories.get().is_empty()}>
                    <p class="empty">"No categories yet. Create your first one above."</p>
                </Show>

                <Show when={move || !loading.get() && !categories.get().is_empty()}>
                    <table class="categories-table">
                        <thead>
                            <tr>
                                <th>"Name"</th>
                                <th>"Path"</th>
                                <th>"Actions"</th>
                            </tr>
                        </thead>
                        <tbody>
                            {move || categories.get().into_iter().map(|cat| {
                                let cat_id = cat.id;
                                view! {
                                    <tr>
                                        <td>{cat.name.clone()}</td>
                                        <td class="category-path">{cat.path.clone()}</td>
                                        <td class="actions">
                                            <button class="btn-danger btn-sm" on:click={move |_| handle_delete(cat_id)}>"Delete"</button>
                                        </td>
                                    </tr>
                                }
                            }).collect_view()}
                        </tbody>
                    </table>
                </Show>
            </div>
        </div>
    }
}
