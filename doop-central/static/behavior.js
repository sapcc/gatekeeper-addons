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

  //This updates the view after a filter or search phrase was set. We will use
  //this in event handlers below.
  let updateView = () => {
    //collect filter settings
    const searchTerms = $("input#search").value.toLowerCase().split(/\s+/);
    const isSelected = {};
    for (const button of $$("div.buttons > button")) {
      isSelected[button.dataset.value] = button.classList.contains("selected");
    }

    //foreach violation...
    for (const violation of $$("ul.violations > li")) {
      //...show only if at least once instance remains on screen
      let hasVisibleInstances = false;
      for (const instance of violation.querySelectorAll(".violation-instance")) {
        const isVisible = isSelected[instance.dataset.layer] && isSelected[instance.dataset.type];
        if (isVisible) {
          hasVisibleInstances = true;
        }
        instance.classList.toggle("hidden", !isVisible);
      }

      //...show only if all search terms are found in the top line
      const text = violation.querySelector(".violation-details").innerText.toLowerCase();
      let matchesSearch = true;
      for (const searchTerm of searchTerms) {
        if (!text.includes(searchTerm)) {
          matchesSearch = false;
          break;
        }
      }

      //apply computed visibility
      violation.classList.toggle("hidden", !(matchesSearch && hasVisibleInstances));
    }

    //hide checks that have all violations hidden
    for (const section of $$("section")) {
      const isHidden = section.querySelectorAll("ul.violations > li:not(.hidden)").length == 0;
      section.classList.toggle("hidden", isHidden);
    }
  };

  //Clear search terms that were carried across reloads.
  $("input#search").value = "";

  //The event handler for the search box is easy.
  $("input#search").addEventListener("input", event => updateView());

  //For the layer/type filters, we need to toggle them manually.
  for (const button of $$("div.buttons > button")) {
    button.addEventListener("click", event => {
      event.preventDefault();
      button.classList.toggle("selected");
      updateView();
    });
  }

})();
