/*******************************************************************************
*
* Copyright 2021 SAP SE
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You should have received a copy of the License along with this
* program. If not, you may obtain a copy of the License at
*
*     http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
*
*******************************************************************************/

(function() {
  var $ = selector => document.querySelector(selector);
  var $$ = selector => document.querySelectorAll(selector);

  //Long lists of violation instances get folded by default. This is the
  //behavior for the unfold button.
  for (const list of $$(".violation-instances")) {
    for (const button of list.querySelectorAll(".unfolder a")) {
      button.addEventListener("click", event => {
        event.preventDefault();
        list.classList.remove("folded");
      });
    }
  }

  //This reads the search/filter controls in the header.
  const getSearchAndFilter = () => {
    const formData = new FormData($("header > form"));
    const searchTerms = (formData.get("search") || "").toLowerCase().split(/\s+/);
    formData.delete("search");
    const filters = [...formData.entries()].filter(pair => pair[1] !== "all");
    return { filters, searchTerms };
  };

  //This helper function takes an `ul.violations > li` and checks if the given
  //search terms appear in its top line (in the `div.violation-details`).
  const doesViolationMatchSearch = (violation, searchTerms) => {
    //optimization: if nothing is searched for, show everything
    if (searchTerms.length === 0) {
      return true;
    }

    //optimization: since `innerText` is quite expensive, cache its result as a data attribute
    const text = (violation.dataset.cachedDetailsText ||= violation.querySelector(".violation-details").innerText.toLowerCase());

    //require all search terms to be present
    return searchTerms.every(term => text.includes(term));
  };

  //This updates the view after a filter or search phrase was set. We will use
  //this in event handlers below.
  const updateView = () => {
    const { filters, searchTerms } = getSearchAndFilter();
    console.log(`Applying ${JSON.stringify({ "filter": Object.fromEntries(filters), "search": searchTerms })}`);

    //foreach violation...
    for (const violation of $$("ul.violations > li")) {
      //...show only if at least once instance remains on screen
      let hasVisibleInstances = false;
      for (const instance of violation.querySelectorAll(".violation-instance")) {
        const isVisible = filters.every(pair => pair[1] === instance.dataset[pair[0]]);
        if (isVisible) {
          hasVisibleInstances = true;
        }
        instance.classList.toggle("hidden", !isVisible);
      }

      //...show only if all search terms are found in the top line
      const matchesSearch = doesViolationMatchSearch(violation, searchTerms);

      //apply computed visibility
      violation.classList.toggle("hidden", !(matchesSearch && hasVisibleInstances));
    }

    //hide checks that have all violations hidden
    for (const section of $$("section.check")) {
      const isHidden = section.querySelectorAll("ul.violations > li:not(.hidden)").length == 0;
      section.classList.toggle("hidden", isHidden);
    }
  };

  //We need to listen on input events to update the view accordingly.
  $("header input").addEventListener("input", event => updateView());
  for (const selector of $$("header select")) {
    selector.addEventListener("change", event => updateView());
  }

  //If search terms were carried across reloads (or entered before this script was loaded), update the view immediately.
  const initial = getSearchAndFilter();
  if (initial.searchTerms.length > 0 || initial.filters.length > 0) {
    updateView();
  }

  //Foldable sections need a click handler to fold/unfold.
  for (const section of $$("section")) {
    section.querySelector("section > h2").addEventListener("click", event => {
      event.preventDefault();
      section.classList.toggle("folded");
    });
  }

})();
