// SPDX-FileCopyrightText: Free Software Foundation Europe <https://fsfe.org>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// templates is a const dict of template names mapping to their CSS properties.
const templates = {
  'pmpc': {
    '--primary': 'var(--fsfe-blue-dark)',
    '--primary-hover': 'var(--fsfe-blue-medium)',
    '--custom-button-color': 'var(--fsfe-green-mix)',
    '--logo-src': 'url(../img/fsfe_logo.png)'
  },
  'ilovefs': {
    '--primary': 'var(--ilovefs-primary)',
    '--primary-hover': 'var(--ilovefs-secondary)',
    '--custom-button-color': 'var(--ilovefs-primary)',
    '--logo-src': 'url(../img/fsfe_logo_ilovefs.png)'
  },
  'sfscon': {
    '--primary': 'var(--sfscon-primary)',
    '--primary-hover': 'var(--sfscon-primary)',
    '--custom-button-color': 'var(--sfscon-primary)',
    '--logo-src': 'url(../img/fsfe_logo.png)'
  }
};

// detailsSelector describes all <details> elements.
const detailsSelector = 'form#sharepicForm > details';

// setTemplate loads CSS variables for a template of the const templates dict.
// Aborts if the template is not part of templates.
const setTemplate = template => {
  if (!templates.hasOwnProperty(template)) {
    return;
  }

  const templateVars = templates[template];
  for (const varKey in templateVars) {
    document.documentElement.style.setProperty(varKey, templateVars[varKey]);
  }
};

// switchTemplateDetail expands the detail view for the given template and
// collides all other template views. Aborts if the template is not part of
// templates.
const switchTemplateDetail = template => {
  if (!templates.hasOwnProperty(template)) {
    return;
  }

  document.querySelectorAll(detailsSelector).forEach((detail) => {
    const isWanted = detail.getAttribute('data-template') == template;
    if (isWanted) {
      detail.setAttribute('open', '');

      const firstTemplate = detail.querySelector('div.grid input');
      firstTemplate.click();
    } else {
      detail.removeAttribute('open');
    }
  });
};

// registerTemplateHandlers activates adjusting themes for the templates when
// clicking resp switching the template groups through their <details>.
const registerTemplateHandlers = () => {
  document.querySelectorAll(detailsSelector).forEach((detail) => {
    const template = detail.getAttribute('data-template');
    const summary = detail.querySelector('summary');

    summary.addEventListener('click', () => {
      event.preventDefault();

      setTemplate(template);
      switchTemplateDetail(template);

      window.location.hash = template;
    });
  });
};

// setTemplateOnLoad if a hash is part of the URL it loads this template.
const setTemplateOnLoad = () => {
  const template = window.location.hash.substr(1);
  if (templates.hasOwnProperty(template)) {
    setTemplate(template);
    switchTemplateDetail(template);
  }
};

setTemplateOnLoad();
registerTemplateHandlers();
