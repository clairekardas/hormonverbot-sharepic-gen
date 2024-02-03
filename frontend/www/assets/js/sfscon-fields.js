// SPDX-FileCopyrightText: Free Software Foundation Europe <https://fsfe.org>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// This piece of JavaScript allows to hide and remove the requirement for the
// message and author description field for those templates having the
// non-standard "no-message" tag.
// The implementation is kind of hackish, but it's late..

document.querySelectorAll("form#sharepicForm > details > div.grid > div").forEach((div) => {
  div.addEventListener('click', () => {
    hidden = div.attributes["no-message"] !== undefined;
    ["label[for=message]", "label[for=authorDesc]"].forEach((selector) => {
      document.querySelector(selector).hidden = hidden;
      document.querySelector(selector + " > *").required = !hidden;
    });
  });
});
