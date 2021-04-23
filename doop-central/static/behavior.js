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

  var $$ = selector => document.querySelectorAll(selector);

  //Long lists of violation instances get folded by default. This is the
  //behavior for the unfold button.
  for (const list of document.querySelectorAll(".violation-instances")) {
    for (const button of list.querySelectorAll(".unfolder a")) {
      button.addEventListener("click", event => {
        event.preventDefault();
        list.classList.remove("folded");
      });
    }
  }

})();
