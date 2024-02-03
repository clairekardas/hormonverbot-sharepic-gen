// SPDX-FileCopyrightText: Free Software Foundation Europe <https://fsfe.org>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// All elements needed in the code.
const responseFormat = document.getElementById('responseFormat');
const resultModal = document.getElementById('resultModal');
const sharepicImg = document.getElementById('sharepicImg');
const sharepicDownloadBtn = document.getElementById('sharepicDownloadBtn');
const sharepicError = document.getElementById('sharepicError');
const sharepicForm = document.getElementById('sharepicForm');
const sharepicFormBtn = document.getElementById('sharepicFormBtn');


// Resets both form button and old modal modifications. Eventually, shows modal.
const switchToModal = () => {
  sharepicFormBtn.setAttribute('aria-busy', 'false');
  sharepicFormBtn.textContent = 'Generate sharepic';

  openModal(resultModal);

  sharepicImg.style.display = 'none';
  sharepicDownloadBtn.style.display = 'none';
  sharepicError.style.display = 'none';
};

// Switch to the modal with a successfully generated sharepic.
const showSharepic = jpeg => {
  switchToModal();

  const imgUrl = 'data:image/jpeg;base64,' + jpeg;

  sharepicImg.setAttribute('src', imgUrl);

  sharepicDownloadBtn.href = imgUrl;
  sharepicDownloadBtn.download = "sharepic.jpg";

  sharepicImg.style.display = '';
  sharepicDownloadBtn.style.display = '';
};

// Switch to the modal with an error message.
const showError = error => {
  switchToModal();

  sharepicError.style.display = '';
  sharepicError.innerHTML = 'Something went wrong:<br /><pre>' + error + '</pre>';
};

// Entry point for sharepic generation.
const composeSharepic = () => {
  sharepicFormBtn.setAttribute('aria-busy', 'true');
  sharepicFormBtn.textContent = 'Please wait…';

  const formData = new FormData(sharepicForm);

  fetch('/sharepic', {
    method: 'POST',
    body: formData,
  })
  .then(resp => resp.json())
  .then(result => {
    if (result.Error == '') {
      showSharepic(result.Jpeg);
    } else {
      showError(result.Error);
    }
  })
  .catch(error => {
    showError(error);
  });
};


// Register the composeSharepic function for form submission.
sharepicForm.addEventListener('submit', event => {
  event.preventDefault();
  composeSharepic();
});


// Request a JSON feedback as the JavaScript frontend is used.
responseFormat.value = 'json';
